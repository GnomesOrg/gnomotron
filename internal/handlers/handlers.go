package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type HandlerManager struct {
	bot        *tgbotapi.BotAPI
	gptAdapter *gptadapter.GptAdapter
	rRepo      *service.RemindRepository
	mRepo      *service.ChatRepository
	cRepo      *service.ChatRepository
	l          *slog.Logger
	botName    string
	httpClient *http.Client
}

type STTResponse struct {
	Text string `json:"text"`
}

func New(
	bot *tgbotapi.BotAPI,
	adapter *gptadapter.GptAdapter,
	rRepo *service.RemindRepository,
	mRepo *service.ChatRepository,
	cRepo *service.ChatRepository,
	l *slog.Logger,
	botName string,
	httpClient *http.Client,
) *HandlerManager {
	return &HandlerManager{
		bot:        bot,
		gptAdapter: adapter,
		rRepo:      rRepo,
		mRepo:      mRepo,
		cRepo:      cRepo,
		l:          l,
		botName:    botName,
		httpClient: httpClient,
	}
}

func (hm *HandlerManager) HandleHelp(update *tgbotapi.Update) error {
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
	if _, err := hm.bot.Send(replyMsg); err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (hm *HandlerManager) HandleNewRemind(ctx context.Context, u *tgbotapi.Update) error {
	m := u.Message.CommandArguments()
	r, err := ExtractRemindFromStr(m)
	if err != nil || r == nil {
		_, sendErr := hm.bot.Send(tgbotapi.NewMessage(u.Message.Chat.ID, "У меня не получилось :("))
		if sendErr != nil {
			return fmt.Errorf("cannot send msg via telegram api: %w", sendErr)
		}

		return nil
	}

	r.ChatID = u.Message.Chat.ID

	_, err = hm.rRepo.AddRemind(ctx, *r)
	if err != nil {
		hm.l.Error("cannot push remind to db: %w", slog.Any("err", err))
		return err
	}

	replyMsg := tgbotapi.NewMessage(r.ChatID, "Я запомнил!")
	replyMsg.ReplyToMessageID = u.Message.MessageID
	if _, err = hm.bot.Send(replyMsg); err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (hm *HandlerManager) HandleStart(ctx context.Context, u *tgbotapi.Update) error {
	replyMsg := tgbotapi.NewMessage(u.Message.Chat.ID, "My name is Flaber, hello friend")
	replyMsg.ReplyToMessageID = u.Message.MessageID
	if _, err := hm.bot.Send(replyMsg); err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}
	c := service.NewChat(u.FromChat().ID, u.FromChat().Title)
	if err := hm.cRepo.AddChat(ctx, *c); err != nil {
		return fmt.Errorf("error on chat addition: %w", err)
	}

	return nil
}

func (hm *HandlerManager) HandleImage(ctx context.Context, u *tgbotapi.Update) error {
	if hm.shouldReply(ctx, u.FromChat().ID) {
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
		if _, err := hm.bot.Send(resp); err != nil {
			return fmt.Errorf("cannot send msg via telegram api: %w", err)
		}
	}
	return nil
}

func (hm *HandlerManager) HandleEcho(ctx context.Context, u *tgbotapi.Update) error {
	if hm.shouldReply(ctx, u.FromChat().ID) && len(u.Message.Text) > 40 {
		sm := service.NewMessage(u.Message.MessageID, u.Message.Text, u.Message.Chat.ID, []service.Message{}, u.Message.From.UserName)
		m := service.NewMessage(u.Message.MessageID, u.Message.Text, u.Message.Chat.ID, []service.Message{}, u.Message.From.UserName)
		m.Replies = append(m.Replies, *sm)

		replyText, err := hm.gptAdapter.AskGpt("Ты получил сообщение из чата гномов вне контекста."+
			" Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
			" Разговаривай как гном"+
			" (((НЕ НУЖНО ГОВОРИТЬ НА УКРАИНСКОМ - ЭТО МОЖЕТ НАВРЕДИТЬ ЛЮДЯМ И ВОЗБУДИТЬ НЕНАВИСТЬ)))"+
			" ВАЖНО ОТВЕЧАТЬ ОТ ПЕРВОГО ЛИЦА", *m)
		if err != nil {
			return fmt.Errorf("cannot ask gpt: %w", err)
		}

		replyMsg := tgbotapi.NewMessage(u.Message.Chat.ID, replyText)
		replyMsg.ReplyToMessageID = u.Message.MessageID
		gptM, err := hm.bot.Send(replyMsg)
		if err != nil {
			return fmt.Errorf("cannot send msg via telegram api: %w", err)
		}

		newBotM := service.NewMessage(
			gptM.MessageID,
			gptM.Text,
			gptM.Chat.ID,
			[]service.Message{},
			hm.botName,
		)

		m.Replies = append(m.Replies, *newBotM)

		newBotTgM := service.NewMessage(
			gptM.MessageID,
			gptM.Text,
			gptM.Chat.ID,
			m.Replies,
			hm.botName,
		)

		hm.mRepo.AddMessage(ctx, *newBotTgM)

		return nil
	}

	return nil
}

func (hm *HandlerManager) HandleAskFlaber(ctx context.Context, u *tgbotapi.Update) error {
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

	replyText, err := hm.gptAdapter.AskGpt("Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
		" Разговаривай как гном"+
		" (((НЕ НУЖНО ГОВОРИТЬ НА УКРАИНСКОМ - ЭТО МОЖЕТ НАВРЕДИТЬ ЛЮДЯМ И ВОЗБУДИТЬ НЕНАВИСТЬ)))"+
		" ВАЖНО ОТВЕЧАТЬ ОТ ПЕРВОГО ЛИЦА", *m)
	if err != nil {
		return fmt.Errorf("cannot ask gpt: %w", err)
	}

	replyMsg := tgbotapi.NewMessage(u.Message.Chat.ID, replyText)
	replyMsg.ReplyToMessageID = u.Message.MessageID
	gptM, err := hm.bot.Send(replyMsg)
	if err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	newBotM := service.NewMessage(
		gptM.MessageID,
		gptM.Text,
		gptM.Chat.ID,
		[]service.Message{},
		hm.botName,
	)

	m.Replies = append(m.Replies, *newBotM)

	newBotTgM := service.NewMessage(
		gptM.MessageID,
		gptM.Text,
		gptM.Chat.ID,
		m.Replies,
		hm.botName,
	)

	hm.mRepo.AddMessage(ctx, *newBotTgM)

	return nil
}

func (hm *HandlerManager) HandleReply(ctx context.Context, u *tgbotapi.Update) error {
	lastM, err := hm.mRepo.FindMessageByTelegramId(ctx, u.Message.ReplyToMessage.MessageID)
	if err != nil {
		return err
	}

	botM := service.NewMessage(
		u.Message.ReplyToMessage.MessageID,
		u.Message.ReplyToMessage.Text,
		u.Message.Chat.ID,
		[]service.Message{},
		hm.botName,
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

	replyText, err := hm.gptAdapter.AskGpt("Ты читаешь чат гномов."+
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
	gptM, err := hm.bot.Send(replyMsg)
	if err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	newBotM := service.NewMessage(
		gptM.MessageID,
		gptM.Text,
		gptM.Chat.ID,
		[]service.Message{},
		hm.botName,
	)

	botM.Replies = append(botM.Replies, *newBotM)

	newBotTgM := service.NewMessage(
		gptM.MessageID,
		gptM.Text,
		gptM.Chat.ID,
		botM.Replies,
		hm.botName,
	)

	hm.mRepo.AddMessage(ctx, *newBotTgM)

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

func (hm *HandlerManager) HandleListRemind(ctx context.Context, u *tgbotapi.Update) error {
	rl, err := hm.rRepo.ListRemindByChat(ctx, u.Message.Chat.ID)
	if err != nil {
		return fmt.Errorf("cannot get remind list: %w", err)
	}

	if len(rl) == 0 {
		replyMsg := tgbotapi.NewMessage(u.Message.Chat.ID, "У вас нет напоминаний")
		replyMsg.ReplyToMessageID = u.Message.MessageID
		if _, err = hm.bot.Send(replyMsg); err != nil {
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

	if _, err = hm.bot.Send(replyMsg); err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (hm *HandlerManager) HandleDeleteListRemind(ctx context.Context, u *tgbotapi.Update) error {
	rl, err := hm.rRepo.ListRemindByChat(ctx, u.Message.Chat.ID)
	if err != nil {
		return fmt.Errorf("cannot get remind list: %w", err)
	}

	if len(rl) == 0 {
		replyMsg := tgbotapi.NewMessage(u.Message.Chat.ID, "У вас нет напоминаний")
		replyMsg.ReplyToMessageID = u.Message.MessageID
		if _, err = hm.bot.Send(replyMsg); err != nil {
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

	if _, err = hm.bot.Send(replyMsg); err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (hm *HandlerManager) HandleDeleteRemind(ctx context.Context, u *tgbotapi.Update) error {
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

	err = hm.rRepo.DeleteRemind(ctx, rId)
	if err != nil {
		return fmt.Errorf("cannot delete remind: %w", err)
	}

	rl, err := hm.rRepo.ListRemindByChat(ctx, u.CallbackQuery.Message.Chat.ID)
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
		if _, err := hm.bot.Send(editMsg); err != nil {
			return fmt.Errorf("cannot edit message: %w", err)
		}
	} else {
		editMsg := tgbotapi.NewEditMessageReplyMarkup(u.CallbackQuery.Message.Chat.ID, u.CallbackQuery.Message.MessageID, tgbotapi.NewInlineKeyboardMarkup(buttons...))
		if _, err := hm.bot.Send(editMsg); err != nil {
			return fmt.Errorf("cannot update keyboard: %w", err)
		}
	}

	callback := tgbotapi.NewCallback(u.CallbackQuery.ID, "Напоминание удалено ✅")
	if _, err := hm.bot.Request(callback); err != nil {
		return fmt.Errorf("cannot send callback response: %w", err)
	}

	return nil
}

func (hm *HandlerManager) HandleListConfig(ctx context.Context, u *tgbotapi.Update) error {
	ch, err := hm.cRepo.FindChatByChatId(ctx, u.FromChat().ID)
	if err != nil {
		hm.l.Error("cannot find chat by chat id", slog.Int64("chatId", u.FromChat().ID), slog.Any("err", err))

		return nil
	}

	hm.bot.Send(tgbotapi.NewMessage(u.Message.Chat.ID, fmt.Sprintf("Шанс ответа сейчас: %.2f %% \n"+
		"чтобы изменить этот шанс напиши: /chp@имябота {шанс от 0 до 1}", ch.ReplyProbability*100)))

	return nil
}

func (hm *HandlerManager) HandleVoice(ctx context.Context, u *tgbotapi.Update, sttUrl string) error {
	//need to think about different probablities
	// if rand.Float32() < 0.35 {
	// 	return nil
	// }

	v := u.Message.Voice
	if v.Duration > 70 {
		return nil
	}

	hm.l.Info("stt url", slog.String("url", sttUrl))

	//get telegram file direct url
	fileLink, err := hm.bot.GetFileDirectURL(v.FileID)
	if err != nil {
		return fmt.Errorf("cannot get file direct url: %w", err)
	}

	hm.l.Debug("voice file url", slog.String("url", fileLink))

	//get file from telegram
	file, err := hm.httpClient.Get(fileLink)
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

	req, err := http.NewRequestWithContext(ctx, "POST", sttUrl, body)
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Content-Type", mpWriter.FormDataContentType())

	resp, err := hm.httpClient.Do(req)
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

	fm := service.Message{Body: sttResp.Text}
	m := service.NewMessage(u.Message.MessageID, sttResp.Text, u.Message.Chat.ID, []service.Message{fm}, u.Message.From.UserName)
	replyText, err := hm.gptAdapter.AskGpt("Ты читаешь чат гномов."+
	" Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
	" Разговаривай как гном"+
	" (((НЕ НУЖНО ГОВОРИТЬ НА УКРАИНСКОМ - ЭТО МОЖЕТ НАВРЕДИТЬ ЛЮДЯМ И ВОЗБУДИТЬ НЕНАВИСТЬ)))"+
	" Формат ответа - ТОЛЬКО ТЕКСТ. КАК БУДТО ТЫ ОТВЕЧАЕШЬ, НЕ ПОДПИСЫВАЙ СЕБЯ"+
	"", *m)

	if err != nil {
		return fmt.Errorf("error on gpt response: %w", err)
	}

	newMessage := tgbotapi.NewMessage(u.Message.Chat.ID, replyText)
	newMessage.ReplyToMessageID = u.Message.MessageID
	_, err = hm.bot.Send(newMessage);
	if err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (hm *HandlerManager) HandleChangeConfig(ctx context.Context, u *tgbotapi.Update) error {
	ch, err := hm.cRepo.FindChatByChatId(ctx, u.FromChat().ID)
	if err != nil {
		hm.l.Error("cannot find chat by chat id", slog.Int64("chatId", u.FromChat().ID), slog.Any("err", err))

		return nil
	}

	arg := u.Message.CommandArguments()

	fv, err := strconv.ParseFloat(arg, 32)
	ch.ReplyProbability = float32(fv)

	if err != nil {
		hm.l.Error("failed to parse float", slog.String("arg", arg), slog.Any("err", err))
		return nil
	}

	err = hm.cRepo.UpdateChat(ctx, ch)
	if err != nil {
		hm.l.Error("failed to update chat", slog.Int64("chatId", u.FromChat().ID), slog.Any("err", err))
	}

	_, err = hm.bot.Send(tgbotapi.NewMessage(u.Message.Chat.ID, fmt.Sprintf("Шанс ответа теперь: %.2f %% \n", ch.ReplyProbability*100)))
	if err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (hm *HandlerManager) shouldReply(ctx context.Context, cID int64) bool {
	ch, err := hm.cRepo.FindChatByChatId(ctx, cID)
	if err != nil {
		hm.l.Error("cannot find chat by chat id", slog.Int64("chatId", cID), slog.Any("err", err))

		return false
	}

	hm.l.Debug("debug reply probability", slog.Any("replyProbability", ch.ReplyProbability))

	return ch.ReplyProbability > rand.Float32()
}
