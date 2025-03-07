package commands

import (
	"context"
	"encoding/json"
	"flabergnomebot/internal/service"
	"fmt"
	"log/slog"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CommandHandler struct {
	l      *slog.Logger
	bot    *tgbotapi.BotAPI
	chRepo *service.ChatRepository
}

func NewCommandHandler(l *slog.Logger, b *tgbotapi.BotAPI, chRepo *service.ChatRepository) *CommandHandler {
	return &CommandHandler{
		bot:    b,
		l:      l,
		chRepo: chRepo,
	}
}

type SendMsgReq struct {
	Msg     string `json:"msg"`
	ChatId  int64  `json:"chatId"`
	ReplyId *int   `json:"replyId"`
}

type SendBcMsgReq struct {
	Msg string `json:"msg"`
}

func (ch *CommandHandler) SendMsgCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	res := &SendMsgReq{}
	err := json.NewDecoder(r.Body).Decode(res)
	if err != nil {
		ch.l.Error("error on decoding request", slog.Any("error", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	m := tgbotapi.NewMessage(res.ChatId, res.Msg)

	if res.ReplyId != nil {
		m.ReplyToMessageID = *res.ReplyId
	}

	ch.l.Info("sending message", slog.String("msg", res.Msg), slog.Any("chatId", res.ChatId), slog.Any("replyId", res.ReplyId))
	_, err = ch.bot.Send(m)
	if err != nil {
		ch.l.Error("error on sending message to telegram", slog.String("error", err.Error()))
	}

	w.WriteHeader(http.StatusCreated)

	fmt.Fprintf(w, "done. msg: %s\n", res.Msg)
}

func (ch *CommandHandler) SendBroadcastMsgCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	res := &SendBcMsgReq{}
	err := json.NewDecoder(r.Body).Decode(res)
	if err != nil {
		ch.l.Error("error on decoding request", slog.Any("error", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	chats, err := ch.chRepo.FindAllChats(context.TODO())
	if err != nil {
		ch.l.Error("error on searching chats", slog.Any("error", err))
	}

	for _, chat := range chats {
		ch.l.Info("sending message", slog.String("msg", res.Msg), slog.Any("chatId", chat.ChatID))
		_, err = ch.bot.Send(tgbotapi.NewMessage(chat.ChatID, res.Msg))
		if err != nil {
			ch.l.Error("error on sending message to telegram", slog.Any("error", err))
		}
	}

	w.WriteHeader(http.StatusCreated)

	fmt.Fprintf(w, "done.")
}
