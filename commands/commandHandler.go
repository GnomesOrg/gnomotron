package commands

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CommandHandler struct {
	bot *tgbotapi.BotAPI
}

func NewCommandHandler(l *slog.Logger, b *tgbotapi.BotAPI) *CommandHandler {
	return &CommandHandler{
		bot: b,
	}
}

type SendMsgReq struct {
	Msg     string `json:"msg"`
	ChatId  int64  `json:"chatId"`
	ReplyId *int    `json:"replyId"`
}

func (ch *CommandHandler) SendMsgCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	res := &SendMsgReq{}
	err := json.NewDecoder(r.Body).Decode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	m := tgbotapi.MessageConfig{}

	if res.ReplyId == nil {
		m = tgbotapi.NewMessage(res.ChatId, res.Msg)
	} else {
		m = tgbotapi.NewMessage(res.ChatId, res.Msg)
		m.ReplyToMessageID = *res.ReplyId
		
	}
	ch.bot.Send(m)
	w.WriteHeader(http.StatusCreated)

	fmt.Fprintf(w, "done. msg: %s\n", res.Msg)
}
