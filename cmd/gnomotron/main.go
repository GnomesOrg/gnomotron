package main

import (
	"context"
	"flabergnomebot/commands"
	"flabergnomebot/internal/config"
	"flabergnomebot/internal/gptadapter"
	"flabergnomebot/internal/handlers"
	"flabergnomebot/internal/service"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	loggerOptions := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	l := slog.New(slog.NewJSONHandler(os.Stdout, loggerOptions))
	cfg := config.LoadConfig()
	adapter := gptadapter.New(cfg.APIKEY, l)
	bot, err := tgbotapi.NewBotAPI(cfg.TOKEN)
	if err != nil {
		l.Error("error on bot init", slog.Any("error", err))
	}

	bot.Debug = true

	//Command server
	ch := commands.NewCommandHandler(bot)

	http.HandleFunc("/sendMsg", ch.SendMsgCommand)
	go http.ListenAndServe(":8055", nil)

	//DB init
	clientOptions := options.Client().ApplyURI(cfg.MONGO_URI)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	client, err := mongo.Connect(ctx, clientOptions)
	defer cancel()
	if err != nil {
		l.Error("error on connect to mongo", slog.Any("error", err))
	}

	rCol := client.Database(cfg.MONGO_DB).Collection(service.RemindCollection)
	mCol := client.Database(cfg.MONGO_DB).Collection(service.MessageCollection)
	remindRepo := service.NewRemindRepository(rCol, l)
	mRepo := service.NewMessageRepository(mCol, l, cfg)
	handlerManager := handlers.NewManager(bot, adapter, remindRepo, mRepo, l)

	botCtx := context.Background()

	//Remind service
	go remindRepo.StartReminderScheduler(bot, botCtx)

	l.Info(fmt.Sprintf("Authorized on account %s", bot.Self.UserName))

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
					l.Info(fmt.Sprintf("[%s] %s", update.Message.From.UserName, update.Message.Text))
					var err error
					switch update.Message.Command() {
					case "start":
						handlerManager.HandleStart(&update)
					case "help":
						err = handlerManager.HandleHelp(&update)
					case "af":
						err = handlerManager.HandleAskFlaber(&update)
					case "nr":
						err = handlerManager.HandleNewRemind(botCtx, &update)
					case "lr":
						err = handlerManager.HandleListRemind(botCtx, &update)
					case "dr":
						err = handlerManager.HandleDeleteRemind(botCtx, &update)
					default:
						if update.Message.ReplyToMessage != nil && update.Message.ReplyToMessage.From.UserName == cfg.BOT_NAME {
							// handle only replies of gnomotron messages

							err = handlerManager.HandleReply(botCtx, &update)
							break
						}

						if update.Message.Photo != nil {
							err = handlerManager.HandleImage(&update)
							break
						}

						if update.Message.Text != "" {
							err = handlerManager.HandleEcho(&update)
							break
						}
					}

					if err != nil {
						l.Error(fmt.Sprintf("error while handling messages: %+v", err))
					}
				}
			}
		}()
	}

	wg.Wait()
}
