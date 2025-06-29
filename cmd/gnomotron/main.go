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
	gcfg := gptadapter.GptConfig{ApiToken: cfg.APIKEY, BotName: cfg.BOT_NAME, BaseURL: cfg.GPT_ENDPOINT}
	adapter := gptadapter.New(l, &gcfg)
	bot, err := tgbotapi.NewBotAPI(cfg.TOKEN)
	if err != nil {
		l.Error("error on bot init", slog.Any("error", err))
	}

	bot.Debug = cfg.BOT_DEBUG

	//DB init
	clientOptions := options.Client().ApplyURI(cfg.MONGO_URI)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	client, err := mongo.Connect(ctx, clientOptions)
	defer cancel()
	if err != nil {
		l.Error("error on connect to mongo", slog.Any("error", err))
	}

	httpCl := http.Client{}
	db := client.Database(cfg.MONGO_DB)
	remindRepo := service.NewRemindRepository(db, l)
	cRepo := service.NewChatRepository(db, l, cfg)

	hc := &handlers.HandlerConfig{
		Bot:        bot,
		GptAdapter: adapter,
		RRepo:      remindRepo,
		CRepo:      cRepo,
		Cfg:        cfg,
		HttpClient: &httpCl,
	}

	handler := handlers.New(hc, l)

	botCtx := context.Background()

	//Remind service
	go remindRepo.StartReminderScheduler(bot, botCtx)

	l.Info(fmt.Sprintf("Authorized on account %s", bot.Self.UserName))

	// Очистка старых обновлений, если бот долго не работал
	oldUpdates, err := bot.GetUpdates(tgbotapi.UpdateConfig{
		Offset: 0,
		Limit:  1,
		Timeout: 0,
	})
	if err != nil {
		l.Error("failed to get latest update", slog.Any("error", err))
	}
	var offset int
	if len(oldUpdates) > 0 {
		offset = oldUpdates[0].UpdateID + 1
	} else {
		offset = 0
	}

	u := tgbotapi.NewUpdate(offset)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	//Command server
	ch := commands.NewCommandHandler(l, bot, cRepo)
	http.HandleFunc("/sendMsg", ch.SendMsgCommand)
	http.HandleFunc("/sendBcMsg", ch.SendBroadcastMsgCommand)
	go http.ListenAndServe(":8055", nil)

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
					err := handler.HandleUpdate(botCtx, &upd)
					if err != nil {
						errMsg := fmt.Sprintf("При выполнении произошла ошибка: %+v", err)
						_, sendErr := bot.Send(tgbotapi.NewMessage(upd.Message.Chat.ID, errMsg))
						if sendErr != nil {
							l.Error(fmt.Sprintf("error while sending message: %+v", err))
						}
					}
				}

				if upd.CallbackQuery != nil {
					l.Debug(fmt.Sprintf("callbackQuery from [%s]: %s", upd.CallbackQuery.From.UserName, upd.CallbackQuery.Data))

					err := handler.HandleDeleteRemind(botCtx, &upd)
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
