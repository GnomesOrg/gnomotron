package main

import (
	"context"
	"flabergnomebot/config"
	"flabergnomebot/service"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg := config.LoadConfig()
	adapter := NewGptAdapter(cfg.APIKEY)
	bot, err := tgbotapi.NewBotAPI(cfg.TOKEN)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	//DB init
	clientOptions := options.Client().ApplyURI(cfg.MONGO_URI)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	client, err := mongo.Connect(ctx, clientOptions)
	defer cancel()
	if err != nil {
		log.Panic(err)
	}

	collection := client.Database(cfg.MONGO_DB).Collection(service.RemindCollection)
	remindRepo := service.NewRemindRepository(client, collection)
	handlerManager := NewHandleManager(bot, adapter, remindRepo)
	//DB init

	remCtx, remCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer remCancel()
	go service.StartReminderScheduler(remindRepo, bot, remCtx)
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// sometimes gnomotron is blocked by previous message, we may handle messages in parralel
	// TODO: graceful shutdown
	// TODO: get workers count from the config
	workersCount := 8
	var wg sync.WaitGroup
	for i := 0; i < workersCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

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
					case "nr":
						handlerManager.HandleNewRemind(&update, remCtx)
					default:
						if update.Message.ReplyToMessage != nil && update.Message.ReplyToMessage.From.UserName == cfg.BOT_NAME {
							// handle only replies of gnomotron messages

							handlerManager.HandleReply(&update)
							continue
						}

						if update.Message.Photo != nil {
							handlerManager.HandleImage(&update)
							continue
						}

						if update.Message.Text != "" {
							handlerManager.HandleEcho(&update)
							continue
						}
					}
				}
			}
		}()
	}

	wg.Wait()
}
