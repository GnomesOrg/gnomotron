package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"math/rand"
	"strconv"
	"strings"
)

type HandlerManager struct {
	bot        *tgbotapi.BotAPI
	gptAdapter *GptAdapter
}

func NewHandleManager(bot *tgbotapi.BotAPI, adapter *GptAdapter) *HandlerManager {
	return &HandlerManager{
		bot:        bot,
		gptAdapter: adapter,
	}
}

func (hm *HandlerManager) HandleHelp(update *tgbotapi.Update) {
	replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "Current chat id is: "+strconv.FormatInt(update.Message.Chat.ID, 10))
	replyMsg.ReplyToMessageID = update.Message.MessageID
	hm.bot.Send(replyMsg)
}

func (hm *HandlerManager) HandleStart(update *tgbotapi.Update) {
	replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "My name is Flaber, hello friend")
	replyMsg.ReplyToMessageID = update.Message.MessageID
	hm.bot.Send(replyMsg)
}

func (hm *HandlerManager) HandleImage(update *tgbotapi.Update) {
	if ShouldReply(0.3) {
		reactions := [...]string{
			"Неприятное изображение",
			"Глупое изображние",
			"Смешное изображние",
			"Философское изображние",
			"Страшное изображние",
		}
		replyText := hm.gptAdapter.AskGpt("Ты гном, говоришь на гномьем языке и отвечаешь от первого лица."+
			" Ты получил изображение. Тебе нужно его прокомментировать.", reactions[rand.Intn(len(reactions))])
		replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
		replyMsg.ReplyToMessageID = update.Message.MessageID
		hm.bot.Send(replyMsg)
	}
}

func (hm *HandlerManager) HandleEcho(update *tgbotapi.Update) {
	if ShouldReply(0.04) && len(update.Message.Text) > 40 {
		replyText := hm.gptAdapter.AskGpt("Ты получил сообщение из чата гномов вне контекста."+
			" Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
			" Разговаривай как гном"+
			" ВАЖНО ОТВЕЧАТЬ ОТ ПЕРВОГО ЛИЦА", update.Message.Text)
		replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
		replyMsg.ReplyToMessageID = update.Message.MessageID
		hm.bot.Send(replyMsg)
	}
}

func (hm *HandlerManager) HandleAskFlaber(update *tgbotapi.Update) {
	replyText := hm.gptAdapter.AskGpt("Тебя заставляют общаться в чате гномов."+
		" Ты ОЧЕНЬ не хочешь отвечать. Но ответ дать ты обязан. Тебе неприятно общаться с гномамы."+
		" Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
		" Разговаривай как гном"+
		" ВАЖНО ОТВЕЧАТЬ ОТ ПЕРВОГО ЛИЦА", strings.TrimPrefix(update.Message.Text, "/af"))
	replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
	replyMsg.ReplyToMessageID = update.Message.MessageID
	hm.bot.Send(replyMsg)
}

//func (hm *HandlerManager) HandleReply(update *tgbotapi.Update) {
//	replyText := hm.gptAdapter.AskGpt("Ты получил сообщение из чата гномов."+
//		" Ты гномик. Отвечай как будто тебя зовут Флабер. Отвечай коротко в один-два предложения."+
//		" Разговаривай как гном", update.Message.Text)
//	replyMsg := tgbotapi.NewMessage(update.Message.Chat.ID, replyText)
//	replyMsg.ReplyToMessageID = update.Message.MessageID
//	hm.bot.Send(replyMsg)
//}

func ShouldReply(probability float32) bool {
	return rand.Float32() < probability
}
