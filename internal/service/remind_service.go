package service

import (
	"context"
	"log"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const RemindCollection = "remind"

type RemindRepository struct {
	client     *mongo.Client
	collection *mongo.Collection
	l          *slog.Logger
}

type Remind struct {
	Id      primitive.ObjectID `bson:"_id,omitempty"`
	Cron    string             `bson:"cron"`
	Message string             `bson:"message"`
	ChatID  int64              `bson:"chat_id"`
}

func NewRemind(c string, m string, chatId int64) *Remind {
	return &Remind{primitive.NewObjectID(), c, m, chatId}
}

func NewRemindRepository(client *mongo.Client, collection *mongo.Collection, l *slog.Logger) *RemindRepository {
	return &RemindRepository{
		client:     client,
		collection: collection,
		l:          l,
	}
}

func (rRepo *RemindRepository) AddRemind(ctx context.Context, r Remind) (*mongo.InsertOneResult, error) {
	rUUID, err := rRepo.collection.InsertOne(ctx, r)
	log.Println(r.Id, r.Message)
	if err != nil {
		return nil, err
	}

	return rUUID, nil
}

func (rRepo *RemindRepository) GetAllReminders(ctx context.Context) ([]Remind, error) {
	var reminders []Remind
	cursor, err := rRepo.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var reminder Remind
		if curErr := cursor.Decode(&reminder); curErr != nil {
			return nil, curErr
		}
		reminders = append(reminders, reminder)
	}

	if curErr := cursor.Err(); curErr != nil {
		return nil, curErr
	}

	return reminders, nil
}

func (rr *RemindRepository) StartReminderScheduler(bot *tgbotapi.BotAPI, ctx context.Context) {
	c := cron.New()

	reminderMap := make(map[primitive.ObjectID]struct{})

	go func() {
		defer c.Stop()
		for {
			rr.l.Debug("trying to fetch reminders")

			reminders, err := rr.GetAllReminders(ctx)
			if err != nil {
				log.Printf("Failed to fetch reminders: %v", err)
				time.Sleep(time.Minute)
				continue
			}

			rr.l.Debug("fetched some reminders", slog.Any("count", len(reminders)))

			for _, reminder := range reminders {

				_, err := c.AddFunc(reminder.Cron, func() {
					msg := tgbotapi.NewMessage(reminder.ChatID, reminder.Message)
					_, err := bot.Send(msg)
					if err != nil {
						log.Printf("Failed to send reminder: %v", err)
					}
				})

				if err != nil {
					log.Printf("Failed to schedule reminder: %v", err)
				} else {
					reminderMap[reminder.Id] = struct{}{}
					rr.l.Debug("reminder added", slog.Any("id", reminder.Id))
				}
			}

			time.Sleep(time.Minute)
		}
	}()

	c.Start()
}
