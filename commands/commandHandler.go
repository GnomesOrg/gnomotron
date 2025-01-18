package commands

import (
	"encoding/json"
	"fmt"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CommandHandler struct {
	bot *tgbotapi.BotAPI
}

func NewCommandHandler(b *tgbotapi.BotAPI) *CommandHandler {
	return &CommandHandler{
		bot: b,
	}
}

type SendMsgReq struct {
	Msg    string `json:"msg"`
	ChatId int64  `json:"chatId"`
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

	ch.bot.Send(tgbotapi.NewMessage(res.ChatId, res.Msg))

	w.WriteHeader(http.StatusCreated)

	fmt.Fprintf(w, "done. msg: %s\n", res.Msg)
}
