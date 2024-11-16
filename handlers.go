package main

import (
	"errors"
	"flabergnomebot/service"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type HandlerManager struct {
	Bot        *tgbotapi.BotAPI
	GptAdapter *GptAdapter
	RemindRep  *service.RemindRepository
}

func NewHandleManager(bot *tgbotapi.BotAPI, adapter *GptAdapter, rRepo *service.RemindRepository) *HandlerManager {
	return &HandlerManager{
		Bot:        bot,
		GptAdapter: adapter,
		RemindRep:  rRepo,
	}
}

func (hm *HandlerManager) HandleHelp(update *tgbotapi.Update) {
	replyMsg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		"Current chat id is: "+strconv.FormatInt(update.Message.Chat.ID, 10)+
			" To create reminder: type"+
			" /nr [{time in crontab format}] {body}",
	)
	replyMsg.ReplyToMessageID = update.Message.MessageID
	hm.Bot.Send(replyMsg)
}

func (hm *HandlerManager) HandleNewRemind(update *tgbotapi.Update) {
	m := update.Message.Text
	r, err := ExtractRemindFromStr(m)
	if err != nil {
		log.Print(err)
		hm.Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "У меня не получилось :("))
	}

	r.ChatID = update.Message.Chat.ID

	hm.RemindRep.AddRemind(*r)

	replyMsg := tgbotapi.NewMessage(r.ChatID, "Я запомнил!")
	hm.Bot.Send(replyMsg)
}

func (hm *HandlerManager) HandleStart(update *tgbotapi.Update) {
	replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "My name is Flaber, hello friend")
	replyMsg.ReplyToMessageID = update.Message.MessageID
	hm.Bot.Send(replyMsg)
}

func (hm *HandlerManager) HandleImage(update *tgbotapi.Update) {
	if isShouldReply(0.3) {
		reactions := [...]string{
			"Неприятное изображение",
			"Глупое изображние",
			"Смешное изображние",
			"Философское изображние",
			"Страшное изображние",
		}
		replyText, err := hm.GptAdapter.AskGpt(
			"Ты гном, говоришь на гномьем языке и отвечаешь от первого лица."+
				" Ты получил изображение. Тебе нужно его прокомментировать.",
			reactions[rand.Intn(len(reactions))],
		)
		if err != nil {
			log.Printf("cannot ask gpt a question: %+v", err)
			return
		}

		replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
		replyMsg.ReplyToMessageID = update.Message.MessageID
		hm.Bot.Send(replyMsg)
	}

	if isShouldReply(0.02) {
		hm.Bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "sperma :)"))
	}
}

func (hm *HandlerManager) HandleEcho(update *tgbotapi.Update) {
	if isShouldReply(0.04) && len(update.Message.Text) > 40 {
		replyText, err := hm.GptAdapter.AskGpt("Ты получил сообщение из чата гномов вне контекста."+
			" Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
			" Разговаривай как гном"+
			" ВАЖНО ОТВЕЧАТЬ ОТ ПЕРВОГО ЛИЦА", update.Message.Text)
		if err != nil {
			log.Printf("cannot ask gpt a question: %+v", err)
			return
		}

		replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
		replyMsg.ReplyToMessageID = update.Message.MessageID
		hm.Bot.Send(replyMsg)
	}
}

func (hm *HandlerManager) HandleAskFlaber(update *tgbotapi.Update) {
	replyText, err := hm.GptAdapter.AskGpt("Тебя заставляют общаться в чате гномов."+
		" Ты ОЧЕНЬ не хочешь отвечать. Но ответ дать ты обязан. Тебе неприятно общаться с гномамы."+
		" Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
		" Разговаривай как гном"+
		" ВАЖНО ОТВЕЧАТЬ ОТ ПЕРВОГО ЛИЦА", strings.TrimPrefix(update.Message.Text, "/af"))
	if err != nil {
		log.Printf("cannot ask gpt a question: %+v", err)
		return
	}

	replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
	replyMsg.ReplyToMessageID = update.Message.MessageID
	hm.Bot.Send(replyMsg)
}

func (hm *HandlerManager) HandleReply(update *tgbotapi.Update) {
	replyText, err := hm.GptAdapter.AskGpt("Ты получил сообщение из чата гномов."+
		" Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
		" Разговаривай как гном", update.Message.Text)
	if err != nil {
		log.Printf("cannot ask gpt a question: %+v", err)
		return
	}

	replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
	replyMsg.ReplyToMessageID = update.Message.MessageID

	// TODO: handle errors propperly, return err here
	if _, err := hm.Bot.Send(replyMsg); err != nil {
		log.Printf("cannot handle reply properly, cannot send message back: %+v", err)
	}
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
