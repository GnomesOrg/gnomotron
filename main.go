package main

import "log"
import "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func main() {

	cfg := LoadConfig()
	adapter := NewGptAdapter(cfg.APIKEY)
	bot, err := tgbotapi.NewBotAPI(cfg.TOKEN)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	handlerManager := NewHandleManager(bot, adapter)

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			switch update.Message.Command() {
			case "start":
				handlerManager.HandleStart(&update)
			case "help":
				handlerManager.HandleHelp(&update)
			case "af":
				handlerManager.HandleAskFlaber(&update)
			default:
				//if update.Message.ReplyToMessage != nil {
				//	handlerManager.HandleReply(&update)
				//	continue
				//}

				if update.Message.Text != "" {
					handlerManager.HandleEcho(&update)
				}

				if update.Message.Photo != nil {
					handlerManager.HandleImage(&update)
				}
			}
		}
	}

}
