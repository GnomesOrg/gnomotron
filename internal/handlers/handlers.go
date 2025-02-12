package handlers

import (
	"context"
	"errors"
	"flabergnomebot/internal/gptadapter"
	"flabergnomebot/internal/service"
	"fmt"
	"log/slog"
	"math/rand"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const replyProbability = 0.02

type HandlerManager struct {
	bot        *tgbotapi.BotAPI
	gptAdapter *gptadapter.GptAdapter
	rRepo      *service.RemindRepository
	mRepo      *service.MessageRepository
	l          *slog.Logger
	botName    string
}

func New(bot *tgbotapi.BotAPI, adapter *gptadapter.GptAdapter, rRepo *service.RemindRepository, mRepo *service.MessageRepository, l *slog.Logger, botName string) *HandlerManager {
	return &HandlerManager{
		bot:        bot,
		gptAdapter: adapter,
		rRepo:      rRepo,
		mRepo:      mRepo,
		l:          l,
		botName:    botName,
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
			"\n\r/dr {remindId}",
	)
	replyMsg.ReplyToMessageID = update.Message.MessageID
	if _, err := hm.bot.Send(replyMsg); err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (hm *HandlerManager) HandleNewRemind(ctx context.Context, u *tgbotapi.Update) error {
	m := u.Message.Text
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

func (hm *HandlerManager) HandleStart(update *tgbotapi.Update) error {
	replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "My name is Flaber, hello friend")
	replyMsg.ReplyToMessageID = update.Message.MessageID
	_, err := hm.bot.Send(replyMsg)
	if err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (hm *HandlerManager) HandleImage(update *tgbotapi.Update) error {
	if isShouldReply(replyProbability) {
		responses := []string{
			"Удали.",
			"ПХАХПАХпхпхаПА",
			"🤓",
			"Я обожаю сиськи",
		}
		randomIndex := rand.Intn(len(responses))
		resp := tgbotapi.NewMessage(update.Message.Chat.ID, responses[randomIndex])
		resp.ReplyToMessageID = update.Message.MessageID
		if _, err := hm.bot.Send(resp); err != nil {
			return fmt.Errorf("cannot send msg via telegram api: %w", err)
		}
	}
	return nil
}

func (hm *HandlerManager) HandleEcho(ctx context.Context, u *tgbotapi.Update) error {
	if isShouldReply(replyProbability) && len(u.Message.Text) > 40 {
		sm := service.NewMessage(u.Message.MessageID, u.Message.Text, u.Message.Chat.ID, []service.Message{}, u.Message.From.UserName)
		m := service.NewMessage(u.Message.MessageID, u.Message.Text, u.Message.Chat.ID, []service.Message{}, u.Message.From.UserName)
		m.Replies = append(m.Replies, *sm)
		replyText, err := hm.gptAdapter.AskGpt("Ты получил сообщение из чата гномов вне контекста."+
			" Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
			" Разговаривай как гном"+
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

func (hm *HandlerManager) HandleAskFlaber(ctx context.Context,u *tgbotapi.Update) error {
	sm := service.NewMessage(u.Message.MessageID, strings.TrimPrefix(u.Message.Text, "/af"), u.Message.Chat.ID, []service.Message{}, u.Message.From.UserName)
	m := service.NewMessage(u.Message.MessageID, strings.TrimPrefix(u.Message.Text, "/af"), u.Message.Chat.ID, []service.Message{}, u.Message.From.UserName)
	m.Replies = append(m.Replies, *sm)
	
	replyText, err := hm.gptAdapter.AskGpt("Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
		" Разговаривай как гном"+
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

	var lastRepl []service.Message
	if lastM != nil {
		lastRepl = lastM.Replies
		botM.Replies = append(botM.Replies, lastRepl...)
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
	if !strings.HasPrefix(input, "/nr") {
		return nil, errors.New("input string must start with /nr")
	}

	re := regexp.MustCompile(`^/nr\s+\[([^\]]+)\]\s+(.+)$`)
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
			fmt.Sprintf("🕒 %s - %s", remind.Cron, remind.Message),
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

func isShouldReply(probability float32) bool {
	return rand.Float32() < probability
}
