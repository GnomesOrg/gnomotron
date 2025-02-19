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
	gcfg := gptadapter.GptConfig{ApiToken: cfg.APIKEY, BotName: cfg.BOT_NAME}
	adapter := gptadapter.New(l, &gcfg)
	bot, err := tgbotapi.NewBotAPI(cfg.TOKEN)
	if err != nil {
		l.Error("error on bot init", slog.Any("error", err))
	}

	bot.Debug = cfg.BOT_DEGUB

	//Command server
	ch := commands.NewCommandHandler(l, bot)

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
	cCol := client.Database(cfg.MONGO_DB).Collection(service.ChatCollection)
	remindRepo := service.NewRemindRepository(rCol, l)
	mRepo := service.NewRepository(mCol, l, cfg)
	cRepo := service.NewRepository(cCol, l, cfg)
	handlerManager := handlers.New(bot, adapter, remindRepo, mRepo, cRepo, l, cfg.BOT_NAME)

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

			for upd := range updates {
				if upd.Message != nil {
					l.Info(
						"new message",
						slog.Int64("chat id", upd.Message.Chat.ID),
						slog.Int("message id", upd.Message.MessageID),
						slog.String("username", upd.Message.From.UserName),
						slog.String("body", upd.Message.Text),
					)
					var err error
					switch upd.Message.CommandWithAt() {
					case "start@" + cfg.BOT_NAME:
						handlerManager.HandleStart(botCtx, &upd)
					case "help@" + cfg.BOT_NAME:
						err = handlerManager.HandleHelp(&upd)
					case "af@" + cfg.BOT_NAME:
						err = handlerManager.HandleAskFlaber(botCtx, &upd)
					case "nr@" + cfg.BOT_NAME:
						err = handlerManager.HandleNewRemind(botCtx, &upd)
					case "lr@" + cfg.BOT_NAME:
						err = handlerManager.HandleListRemind(botCtx, &upd)
					case "dr@" + cfg.BOT_NAME:
						err = handlerManager.HandleDeleteListRemind(botCtx, &upd)
					default:
						if upd.Message.ReplyToMessage != nil && upd.Message.ReplyToMessage.From.UserName == cfg.BOT_NAME {
							// handle only replies of gnomotron messages

							err = handlerManager.HandleReply(botCtx, &upd)
							break
						}

						if upd.Message.Photo != nil {
							err = handlerManager.HandleImage(&upd)
							break
						}

						if upd.Message.Text != "" {
							err = handlerManager.HandleEcho(botCtx, &upd)
							break
						}
					}

					if err != nil {
						l.Error(fmt.Sprintf("error while handling messages: %+v", err))
					}
				}

				if upd.CallbackQuery != nil {
					l.Debug(fmt.Sprintf("callbackQuery from [%s]: %s", upd.CallbackQuery.From.UserName, upd.CallbackQuery.Data))

					err := handlerManager.HandleDeleteRemind(botCtx, &upd)
					if err != nil {
						l.Error(fmt.Sprintf("error while handling callback query: %+v", err))
					}

					callback := tgbotapi.NewCallback(upd.CallbackQuery.ID, "Обработано")
					if _, err := bot.Request(callback); err != nil {
						l.Error(fmt.Sprintf("error sending callback response: %+v", err))
					}
				}
			}
		}()
	}

	wg.Wait()
}
