package service

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"time"
)

const RemindCollection = "remind"

type RemindRepository struct {
	client     *mongo.Client
	collection *mongo.Collection
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

func NewRemindRepository(client *mongo.Client, collection *mongo.Collection) *RemindRepository {
	return &RemindRepository{client, collection}
}

func (rRepo *RemindRepository) AddRemind(r Remind) *mongo.InsertOneResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rUUID, err := rRepo.collection.InsertOne(ctx, r)
	if err != nil {
		log.Printf("Can't push reminder to db: %+v", err)
	}

	return rUUID
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
		if err := cursor.Decode(&reminder); err != nil {
			return nil, err
		}
		reminders = append(reminders, reminder)
	}

	return reminders, nil
}

func StartReminderScheduler(remindRepo *RemindRepository, bot *tgbotapi.BotAPI) {
	c := cron.New()
	ctx := context.Background()

	go func() {
		for {
			reminders, err := remindRepo.GetAllReminders(ctx)
			if err != nil {
				log.Printf("Failed to fetch reminders: %v", err)
				time.Sleep(time.Minute)
				continue
			}

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
				}
			}

			time.Sleep(15 * time.Minute)
		}
	}()

	c.Start()
}
