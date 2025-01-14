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
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const replyProbability = 0.02

type HandlerManager struct {
	bot        *tgbotapi.BotAPI
	gptAdapter *gptadapter.GptAdapter
	remindRep  *service.RemindRepository
	l          *slog.Logger
}

func NewManager(bot *tgbotapi.BotAPI, adapter *gptadapter.GptAdapter, rRepo *service.RemindRepository, l *slog.Logger) *HandlerManager {
	return &HandlerManager{
		bot:        bot,
		gptAdapter: adapter,
		remindRep:  rRepo,
		l:          l,
	}
}

func (hm *HandlerManager) HandleHelp(update *tgbotapi.Update) error {
	replyMsg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		"Current chat id is: "+strconv.FormatInt(update.Message.Chat.ID, 10)+
			" To create reminder: type"+
			" /nr [{time in crontab format}] {body}",
	)
	replyMsg.ReplyToMessageID = update.Message.MessageID
	if _, err := hm.bot.Send(replyMsg); err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (hm *HandlerManager) HandleNewRemind(update *tgbotapi.Update, ctx context.Context) error {
	m := update.Message.Text
	r, err := ExtractRemindFromStr(m)
	if err != nil {
		_, sendErr := hm.bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "У меня не получилось :("))
		if sendErr != nil {
			return fmt.Errorf("cannot send msg via telegram api: %w", sendErr)
		}
	}

	r.ChatID = update.Message.Chat.ID

	hm.remindRep.AddRemind(ctx, *r)

	replyMsg := tgbotapi.NewMessage(r.ChatID, "Я запомнил!")
	hm.bot.Send(replyMsg)

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
	if isShouldReply(0.03) {
		reactions := [...]string{
			"Неприятное изображение",
			"Глупое изображение",
			"Смешное изображение",
			"Философское изображение",
			"Страшное изображение",
		}
		replyText, err := hm.gptAdapter.AskGpt(
			"Ты гном, говоришь на гномьем языке и отвечаешь от первого лица."+
				" Ты получил изображение. Тебе нужно его коротко прокомментировать.",
			reactions[rand.Intn(len(reactions))],
		)
		if err != nil {
			return fmt.Errorf("cannot ask gpt: %w", err)
		}

		replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
		replyMsg.ReplyToMessageID = update.Message.MessageID
		if _, err = hm.bot.Send(replyMsg); err != nil {
			return fmt.Errorf("cannot send msg via telegram api: %w", err)
		}
	}

	if isShouldReply(replyProbability) {
		if _, err := hm.bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "siski :)")); err != nil {
			return fmt.Errorf("cannot send msg via telegram api: %w", err)
		}
	}

	return nil
}

func (hm *HandlerManager) HandleEcho(update *tgbotapi.Update) error {
	if isShouldReply(replyProbability) && len(update.Message.Text) > 40 {
		replyText, err := hm.gptAdapter.AskGpt("Ты получил сообщение из чата гномов вне контекста."+
			" Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
			" Разговаривай как гном"+
			" ВАЖНО ОТВЕЧАТЬ ОТ ПЕРВОГО ЛИЦА", update.Message.Text)
		if err != nil {
			return fmt.Errorf("cannot ask gpt: %w", err)
		}

		replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
		replyMsg.ReplyToMessageID = update.Message.MessageID
		if _, err = hm.bot.Send(replyMsg); err != nil {
			return fmt.Errorf("cannot send msg via telegram api: %w", err)
		}
	}

	return nil
}

func (hm *HandlerManager) HandleAskFlaber(update *tgbotapi.Update) error {
	replyText, err := hm.gptAdapter.AskGpt("Тебя заставляют общаться в чате гномов."+
		" Ты ОЧЕНЬ не хочешь отвечать. Но ответ дать ты обязан. Тебе неприятно общаться с гномамы."+
		" Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
		" Разговаривай как гном"+
		" ВАЖНО ОТВЕЧАТЬ ОТ ПЕРВОГО ЛИЦА", strings.TrimPrefix(update.Message.Text, "/af"))
	if err != nil {
		return fmt.Errorf("cannot ask gpt: %w", err)
	}

	replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
	replyMsg.ReplyToMessageID = update.Message.MessageID
	_, err = hm.bot.Send(replyMsg)

	if err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

	return nil
}

func (hm *HandlerManager) HandleReply(update *tgbotapi.Update) error {
	replyText, err := hm.gptAdapter.AskGpt("Ты получил сообщение из чата гномов."+
		" Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
		" Разговаривай как гном", update.Message.Text)
	if err != nil {
		return fmt.Errorf("cannot ask gpt: %w", err)
	}

	replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
	replyMsg.ReplyToMessageID = update.Message.MessageID
	if _, err := hm.bot.Send(replyMsg); err != nil {
		return fmt.Errorf("cannot send msg via telegram api: %w", err)
	}

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

func isShouldReply(probability float32) bool {
	return rand.Float32() < probability
}
