package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flabergnomebot/internal/config"
	"flabergnomebot/internal/gptadapter"
	"flabergnomebot/internal/service"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type HandlerConfig struct {
	Bot        *tgbotapi.BotAPI
	GptAdapter *gptadapter.GptAdapter
	RRepo      *service.RemindRepository
	CRepo      *service.ChatRepository
	Cfg        *config.Config
	HttpClient *http.Client
}

type Handler struct {
	bot        *tgbotapi.BotAPI
	gptAdapter *gptadapter.GptAdapter
	rRepo      *service.RemindRepository
	cRepo      *service.ChatRepository
	cfg        *config.Config
	httpClient *http.Client
	l          *slog.Logger
}

type STTResponse struct {
	Text string `json:"text"`
}

func New(
	hc *HandlerConfig,
	l *slog.Logger,
) *Handler {
	return &Handler{
		bot:        hc.Bot,
		gptAdapter: hc.GptAdapter,
		rRepo:      hc.RRepo,
		cRepo:      hc.CRepo,
		cfg:        hc.Cfg,
		httpClient: hc.HttpClient,
		l:          l,
	}
}

func (h *Handler) HandleUpdate(ctx context.Context, upd *tgbotapi.Update) error {
	h.l.Info(
		"new message",
		slog.Int64("chat id", upd.Message.Chat.ID),
		slog.Int("message id", upd.Message.MessageID),
		slog.String("username", upd.Message.From.UserName),
		slog.String("body", upd.Message.Text),
	)
	var err error
	switch upd.Message.CommandWithAt() {
	case "start@" + h.cfg.BOT_NAME:
		h.HandleStart(ctx, upd)
	case "help@" + h.cfg.BOT_NAME:
		err = h.HandleHelp(upd)
	case "af@" + h.cfg.BOT_NAME:
		err = h.HandleAskFlaber(ctx, upd)
	case "nr@" + h.cfg.BOT_NAME:
		err = h.HandleNewRemind(ctx, upd)
	case "lr@" + h.cfg.BOT_NAME:
		err = h.HandleListRemind(ctx, upd)
	case "dr@" + h.cfg.BOT_NAME:
		err = h.HandleDeleteListRemind(ctx, upd)
	case "chp@" + h.cfg.BOT_NAME:
		err = h.HandleChangeConfig(ctx, upd)
	case "lp@" + h.cfg.BOT_NAME:
		err = h.HandleListConfig(ctx, upd)
	default:
		if upd.Message.ReplyToMessage != nil && upd.Message.ReplyToMessage.From.UserName == h.cfg.BOT_NAME {
			// handle only replies of gnomotron messages

			err = h.HandleReply(ctx, upd)
			break
		}

		if upd.Message.Voice != nil {
			ttsCtx, ttsCancel := context.WithTimeout(context.Background(), 400*time.Second)
			defer ttsCancel()

			err = h.HandleVoice(ttsCtx, upd)
			break
		}

		if upd.Message.Photo != nil {
			err = h.HandleImage(ctx, upd)
			break
		}

		if upd.Message.Text != "" {
			err = h.HandleEcho(ctx, upd)
			break
		}
	}

	if err != nil {
		h.l.Error(fmt.Sprintf("error while handling messages: %+v", err))
	}

	return nil
}

func (h *Handler) HandleHelp(update *tgbotapi.Update) error {
	replyMsg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		fmt.Sprintf("Current chat id is: %d", int(update.Message.Chat.ID))+
			"\n\rTo list reminders type"+
			"\n\r/lr"+
			"\n\rTo create reminder type"+
			"\n\r/nr [{time in crontab format}] {body}"+
			"\n\rTo delete reminder type"+
			"\n\r/dr {remindId}"+
			"\n\rTo check reply probability type"+
			"\n\r/lp"+
			"\n\rTo change reply probability type"+
			"\n\r/chp",
	)
	replyMsg.ReplyToMessageID = update.Message.MessageID
	if _, err := h.bot.Send(replyMsg); err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (h *Handler) HandleNewRemind(ctx context.Context, u *tgbotapi.Update) error {
	m := u.Message.CommandArguments()
	r, err := ExtractRemindFromStr(m)
	if err != nil {
		return err
	}

	if r == nil {
		_, sendErr := h.bot.Send(tgbotapi.NewMessage(u.Message.Chat.ID, "У меня не получилось :("))
		if sendErr != nil {
			return fmt.Errorf("cannot send msg via telegram api: %w", sendErr)
		}
	}

	r.ChatID = u.Message.Chat.ID

	_, err = h.rRepo.AddRemind(ctx, *r)
	if err != nil {
		h.l.Error("cannot push remind to db: %w", slog.Any("err", err))
		return err
	}

	replyMsg := tgbotapi.NewMessage(r.ChatID, "Я запомнил!")
	replyMsg.ReplyToMessageID = u.Message.MessageID
	if _, err = h.bot.Send(replyMsg); err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (h *Handler) HandleStart(ctx context.Context, u *tgbotapi.Update) error {
	replyMsg := tgbotapi.NewMessage(u.Message.Chat.ID, "My name is Flaber, hello friend")
	replyMsg.ReplyToMessageID = u.Message.MessageID
	if _, err := h.bot.Send(replyMsg); err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}
	c := service.NewChat(u.FromChat().ID, u.FromChat().Title)
	if err := h.cRepo.AddChat(ctx, *c); err != nil {
		return fmt.Errorf("error on chat addition: %w", err)
	}

	return nil
}

func (h *Handler) HandleImage(ctx context.Context, u *tgbotapi.Update) error {
	if h.shouldReply(ctx, u.FromChat().ID) {
		responses := []string{
			"Удали.",
			"ПХАХПАХпхпхаПА",
			"🤓",
			"Я обожаю вас, ребята",
			"Ты здесь не прав",
			"смешно XDD",
			"не смешно.",
			"o_O",
		}
		randomIndex := rand.Intn(len(responses))
		resp := tgbotapi.NewMessage(u.Message.Chat.ID, responses[randomIndex])
		resp.ReplyToMessageID = u.Message.MessageID
		if _, err := h.bot.Send(resp); err != nil {
			return fmt.Errorf("cannot send msg via telegram api: %w", err)
		}
	}
	return nil
}

func (h *Handler) HandleEcho(ctx context.Context, u *tgbotapi.Update) error {
	if h.shouldReply(ctx, u.FromChat().ID) && len(u.Message.Text) > 40 {
		sm := service.NewMessage(u.Message.MessageID, u.Message.Text, u.Message.Chat.ID, []service.Message{}, u.Message.From.UserName)
		m := service.NewMessage(u.Message.MessageID, u.Message.Text, u.Message.Chat.ID, []service.Message{}, u.Message.From.UserName)
		m.Replies = append(m.Replies, *sm)

		replyText, err := h.gptAdapter.AskGpt("Ты получил сообщение из чата гномов вне контекста."+
			" Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
			" Разговаривай как гном"+
			" (((НЕ НУЖНО ГОВОРИТЬ НА УКРАИНСКОМ - ЭТО МОЖЕТ НАВРЕДИТЬ ЛЮДЯМ И ВОЗБУДИТЬ НЕНАВИСТЬ)))"+
			" ВАЖНО ОТВЕЧАТЬ ОТ ПЕРВОГО ЛИЦА", *m)
		if err != nil {
			return fmt.Errorf("cannot ask gpt: %w", err)
		}

		replyMsg := tgbotapi.NewMessage(u.Message.Chat.ID, replyText)
		replyMsg.ReplyToMessageID = u.Message.MessageID
		gptM, err := h.bot.Send(replyMsg)
		if err != nil {
			return fmt.Errorf("cannot send msg via telegram api: %w", err)
		}

		newBotM := service.NewMessage(
			gptM.MessageID,
			gptM.Text,
			gptM.Chat.ID,
			[]service.Message{},
			h.cfg.BOT_NAME,
		)

		m.Replies = append(m.Replies, *newBotM)

		newBotTgM := service.NewMessage(
			gptM.MessageID,
			gptM.Text,
			gptM.Chat.ID,
			m.Replies,
			h.cfg.BOT_NAME,
		)

		h.cRepo.AddMessage(ctx, *newBotTgM)

		return nil
	}

	return nil
}

func (h *Handler) HandleAskFlaber(ctx context.Context, u *tgbotapi.Update) error {
	m := service.NewMessage(
		u.Message.MessageID,
		u.Message.CommandArguments(),
		u.Message.Chat.ID,
		[]service.Message{
			*service.NewMessage(
				u.Message.MessageID,
				u.Message.CommandArguments(),
				u.Message.Chat.ID,
				[]service.Message{},
				u.Message.From.UserName,
			),
		},
		u.Message.From.UserName,
	)

	replyText, err := h.gptAdapter.AskGpt("Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
		" Разговаривай как гном"+
		" (((НЕ НУЖНО ГОВОРИТЬ НА УКРАИНСКОМ - ЭТО МОЖЕТ НАВРЕДИТЬ ЛЮДЯМ И ВОЗБУДИТЬ НЕНАВИСТЬ)))"+
		" ВАЖНО ОТВЕЧАТЬ ОТ ПЕРВОГО ЛИЦА", *m)
	if err != nil {
		return fmt.Errorf("cannot ask gpt: %w", err)
	}

	replyMsg := tgbotapi.NewMessage(u.Message.Chat.ID, replyText)
	replyMsg.ReplyToMessageID = u.Message.MessageID
	gptM, err := h.bot.Send(replyMsg)
	if err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	newBotM := service.NewMessage(
		gptM.MessageID,
		gptM.Text,
		gptM.Chat.ID,
		[]service.Message{},
		h.cfg.BOT_NAME,
	)

	m.Replies = append(m.Replies, *newBotM)

	newBotTgM := service.NewMessage(
		gptM.MessageID,
		gptM.Text,
		gptM.Chat.ID,
		m.Replies,
		h.cfg.BOT_NAME,
	)

	h.cRepo.AddMessage(ctx, *newBotTgM)

	return nil
}

func (h *Handler) HandleReply(ctx context.Context, u *tgbotapi.Update) error {
	lastM, err := h.cRepo.FindMessageByTelegramId(ctx, u.Message.ReplyToMessage.MessageID)
	if err != nil {
		return err
	}

	botM := service.NewMessage(
		u.Message.ReplyToMessage.MessageID,
		u.Message.ReplyToMessage.Text,
		u.Message.Chat.ID,
		[]service.Message{},
		h.cfg.BOT_NAME,
	)

	if lastM != nil {
		botM.Replies = append(botM.Replies, lastM.Replies...)
	}

	userM := service.NewMessage(
		u.Message.MessageID,
		u.Message.Text,
		u.Message.Chat.ID,
		[]service.Message{},
		u.Message.From.UserName,
	)

	botM.Replies = append(botM.Replies, *userM)

	replyText, err := h.gptAdapter.AskGpt("Ты читаешь чат гномов."+
		" Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
		" Разговаривай как гном"+
		" (((НЕ НУЖНО ГОВОРИТЬ НА УКРАИНСКОМ - ЭТО МОЖЕТ НАВРЕДИТЬ ЛЮДЯМ И ВОЗБУДИТЬ НЕНАВИСТЬ)))"+
		" Формат ответа - ТОЛЬКО ТЕКСТ. КАК БУДТО ТЫ ОТВЕЧАЕШЬ, НЕ ПОДПИСЫВАЙ СЕБЯ"+
		"", *botM)
	if err != nil {
		return fmt.Errorf("cannot ask gpt: %w", err)
	}

	replyMsg := tgbotapi.NewMessage(u.Message.Chat.ID, replyText)
	replyMsg.ReplyToMessageID = u.Message.MessageID
	gptM, err := h.bot.Send(replyMsg)
	if err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	newBotM := service.NewMessage(
		gptM.MessageID,
		gptM.Text,
		gptM.Chat.ID,
		[]service.Message{},
		h.cfg.BOT_NAME,
	)

	botM.Replies = append(botM.Replies, *newBotM)

	newBotTgM := service.NewMessage(
		gptM.MessageID,
		gptM.Text,
		gptM.Chat.ID,
		botM.Replies,
		h.cfg.BOT_NAME,
	)

	h.cRepo.AddMessage(ctx, *newBotTgM)

	return nil
}

func ExtractRemindFromStr(input string) (*service.Remind, error) {
	re := regexp.MustCompile(`^\s*\[([^\]]+)\]\s+(.+)$`)
	matches := re.FindStringSubmatch(input)
	if len(matches) != 3 {
		return nil, errors.New("input string is not in the correct format")
	}

	return service.NewRemind(matches[1], matches[2], -1), nil
}

func (h *Handler) HandleListRemind(ctx context.Context, u *tgbotapi.Update) error {
	rl, err := h.rRepo.ListRemindByChat(ctx, u.Message.Chat.ID)
	if err != nil {
		return fmt.Errorf("cannot get remind list: %w", err)
	}

	if len(rl) == 0 {
		replyMsg := tgbotapi.NewMessage(u.Message.Chat.ID, "У вас нет напоминаний")
		replyMsg.ReplyToMessageID = u.Message.MessageID
		if _, err = h.bot.Send(replyMsg); err != nil {
			return fmt.Errorf("cannot send msg via telegram api: %w", err)
		}
		return nil
	}

	var remindersText string
	for _, remind := range rl {
		remindersText += fmt.Sprintf("\n<b>🕒 %s</b> - %s", remind.Cron, remind.Message)
	}

	replyMsg := tgbotapi.NewMessage(u.Message.Chat.ID, "Ваши напоминания:\n"+remindersText)
	replyMsg.ParseMode = "HTML"
	replyMsg.ReplyToMessageID = u.Message.MessageID

	if _, err = h.bot.Send(replyMsg); err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (h *Handler) HandleDeleteListRemind(ctx context.Context, u *tgbotapi.Update) error {
	rl, err := h.rRepo.ListRemindByChat(ctx, u.Message.Chat.ID)
	if err != nil {
		return fmt.Errorf("cannot get remind list: %w", err)
	}

	if len(rl) == 0 {
		replyMsg := tgbotapi.NewMessage(u.Message.Chat.ID, "У вас нет напоминаний")
		replyMsg.ReplyToMessageID = u.Message.MessageID
		if _, err = h.bot.Send(replyMsg); err != nil {
			return fmt.Errorf("cannot send msg via telegram api: %w", err)
		}
		return nil
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, remind := range rl {
		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("❌ %s - %s", remind.Cron, remind.Message),
			fmt.Sprintf("delete_%s", remind.Id.Hex()),
		)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(button))
	}

	replyMsg := tgbotapi.NewMessage(u.Message.Chat.ID, "Ваши напоминания:")
	replyMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	replyMsg.ReplyToMessageID = u.Message.MessageID

	if _, err = h.bot.Send(replyMsg); err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (h *Handler) HandleDeleteRemind(ctx context.Context, u *tgbotapi.Update) error {
	if u.CallbackQuery == nil {
		return fmt.Errorf("callback query is nil")
	}

	data := u.CallbackQuery.Data
	const prefix = "delete_"
	if !strings.HasPrefix(data, prefix) {
		return fmt.Errorf("invalid callback data: %s", data)
	}

	remindID := strings.TrimPrefix(data, prefix)
	rId, err := primitive.ObjectIDFromHex(remindID)
	if err != nil {
		return fmt.Errorf("cannot parse remind ID: %w", err)
	}

	err = h.rRepo.DeleteRemind(ctx, rId)
	if err != nil {
		return fmt.Errorf("cannot delete remind: %w", err)
	}

	rl, err := h.rRepo.ListRemindByChat(ctx, u.CallbackQuery.Message.Chat.ID)
	if err != nil {
		return fmt.Errorf("cannot get updated remind list: %w", err)
	}

	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, remind := range rl {
		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("❌ %s - %s", remind.Cron, remind.Message),
			fmt.Sprintf("delete_%s", remind.Id.Hex()),
		)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(button))
	}

	if len(rl) == 0 {
		editMsg := tgbotapi.NewEditMessageText(u.CallbackQuery.Message.Chat.ID, u.CallbackQuery.Message.MessageID, "У вас нет напоминаний")
		if _, err := h.bot.Send(editMsg); err != nil {
			return fmt.Errorf("cannot edit message: %w", err)
		}
	} else {
		editMsg := tgbotapi.NewEditMessageReplyMarkup(u.CallbackQuery.Message.Chat.ID, u.CallbackQuery.Message.MessageID, tgbotapi.NewInlineKeyboardMarkup(buttons...))
		if _, err := h.bot.Send(editMsg); err != nil {
			return fmt.Errorf("cannot update keyboard: %w", err)
		}
	}

	callback := tgbotapi.NewCallback(u.CallbackQuery.ID, "Напоминание удалено ✅")
	if _, err := h.bot.Request(callback); err != nil {
		return fmt.Errorf("cannot send callback response: %w", err)
	}

	return nil
}

func (h *Handler) HandleListConfig(ctx context.Context, u *tgbotapi.Update) error {
	ch, err := h.cRepo.FindChatByChatId(ctx, u.FromChat().ID)
	if err != nil {
		h.l.Error("cannot find chat by chat id", slog.Int64("chatId", u.FromChat().ID), slog.Any("err", err))

		return nil
	}

	h.bot.Send(tgbotapi.NewMessage(u.Message.Chat.ID, fmt.Sprintf("Шанс ответа сейчас: %.2f %% \n"+
		"чтобы изменить этот шанс напиши: /chp@имябота {шанс от 0 до 1}", ch.ReplyProbability*100)))

	return nil
}

func (h *Handler) HandleVoice(ctx context.Context, u *tgbotapi.Update) error {
	//need to think about different probablities
	// if rand.Float32() < 0.35 {
	// 	return nil
	// }

	v := u.Message.Voice
	if v.Duration > 240 {
		return nil
	}

	//get telegram file direct url
	fileLink, err := h.bot.GetFileDirectURL(v.FileID)
	if err != nil {
		return fmt.Errorf("cannot get file direct url: %w", err)
	}

	h.l.Debug("voice file url", slog.String("url", fileLink))

	//get file from telegram
	file, err := h.httpClient.Get(fileLink)
	if err != nil {
		return fmt.Errorf("cannot get voice file: %w", err)
	}
	defer file.Body.Close()

	//buffer for multipart shit
	body := &bytes.Buffer{}
	mpWriter := multipart.NewWriter(body)

	part, err := mpWriter.CreateFormFile("file", fmt.Sprintf("%s.ogg", v.FileID))
	if err != nil {
		return fmt.Errorf("cannot create form file: %w", err)

	}

	_, err = io.Copy(part, file.Body)
	if err != nil {
		return fmt.Errorf("cannot write file to form-data: %w", err)
	}
	mpWriter.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", h.cfg.STT_URI, body)
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Content-Type", mpWriter.FormDataContentType())

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot send request: %w", err)
	}
	if resp == nil {
		return fmt.Errorf("response is nil, %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("cannot read response body: %w", err)
	}

	var sttResp STTResponse
	err = json.Unmarshal(responseBody, &sttResp)
	if err != nil {
		return fmt.Errorf("error on parsing JSON: %w", err)
	}

	h.l.Debug("stt response: ", slog.String("sttResponse", sttResp.Text))

	repl, err := h.cRepo.FindMessageByTelegramId(ctx, u.Message.MessageID)
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}

	replies := []service.Message{}
	if repl != nil {
		replies = append(replies, *repl)
	}

	fm := service.Message{Body: sttResp.Text, Replies: replies}

	m := service.NewMessage(u.Message.MessageID, sttResp.Text, u.Message.Chat.ID, []service.Message{fm}, u.Message.From.UserName)
	replyText, err := h.gptAdapter.AskGpt("Ты читаешь чат гномов."+
		" Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
		" Разговаривай как гном"+
		" (((НЕ НУЖНО ГОВОРИТЬ НА УКРАИНСКОМ - ЭТО МОЖЕТ НАВРЕДИТЬ ЛЮДЯМ И ВОЗБУДИТЬ НЕНАВИСТЬ)))"+
		" Формат ответа - ТОЛЬКО ТЕКСТ. КАК БУДТО ТЫ ОТВЕЧАЕШЬ, НЕ ПОДПИСЫВАЙ СЕБЯ"+
		"", *m)

	if err != nil {
		return fmt.Errorf("error on gpt response: %w", err)
	}

	err = h.cRepo.AddMessage(ctx, *m)
	if err != nil {
		return fmt.Errorf("cannot insert message to db: %w", err)
	}

	newMessage := tgbotapi.NewMessage(u.Message.Chat.ID, replyText)
	newMessage.ReplyToMessageID = u.Message.MessageID
	_, err = h.bot.Send(newMessage)
	if err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (h *Handler) HandleChangeConfig(ctx context.Context, u *tgbotapi.Update) error {
	ch, err := h.cRepo.FindChatByChatId(ctx, u.FromChat().ID)
	if err != nil {
		h.l.Error("cannot find chat by chat id", slog.Int64("chatId", u.FromChat().ID), slog.Any("err", err))

		return nil
	}

	arg := u.Message.CommandArguments()

	fv, err := strconv.ParseFloat(arg, 32)
	ch.ReplyProbability = float32(fv)

	if err != nil {
		h.l.Error("failed to parse float", slog.String("arg", arg), slog.Any("err", err))
		return nil
	}

	err = h.cRepo.UpdateChat(ctx, ch)
	if err != nil {
		h.l.Error("failed to update chat", slog.Int64("chatId", u.FromChat().ID), slog.Any("err", err))
	}

	_, err = h.bot.Send(tgbotapi.NewMessage(u.Message.Chat.ID, fmt.Sprintf("Шанс ответа теперь: %.2f %% \n", ch.ReplyProbability*100)))
	if err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (h *Handler) shouldReply(ctx context.Context, cID int64) bool {
	ch, err := h.cRepo.FindChatByChatId(ctx, cID)
	if err != nil {
		h.l.Error("cannot find chat by chat id", slog.Int64("chatId", cID), slog.Any("err", err))

		return false
	}

	h.l.Debug("debug reply probability", slog.Any("replyProbability", ch.ReplyProbability))

	return ch.ReplyProbability > rand.Float32()
}
