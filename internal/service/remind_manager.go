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
	collection *mongo.Collection
	l          *slog.Logger
	rMap       map[primitive.ObjectID]cron.EntryID
	crS        []cron.EntryID
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

func NewRemindRepository(collection *mongo.Collection, l *slog.Logger) *RemindRepository {
	return &RemindRepository{
		collection: collection,
		l:          l,
		rMap:       make(map[primitive.ObjectID]cron.EntryID),
		crS:        make([]cron.EntryID, 0),
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

	go func() {
		defer c.Stop()
		for {
			rr.l.Debug("trying to fetch reminders")

			reminders, err := rr.GetAllReminders(ctx)
			if err != nil {
				rr.l.Error("Failed to fetch reminders", slog.Any("error", err))
				time.Sleep(time.Minute)
				continue
			}

			rr.l.Debug("fetched some reminders", slog.Any("count", len(reminders)))

			for _, reminder := range reminders {
				if _, exists := rr.rMap[reminder.Id]; exists {
					continue
				}

				crId, err := c.AddFunc(reminder.Cron, func() {
					msg := tgbotapi.NewMessage(reminder.ChatID, reminder.Message)
					_, err := bot.Send(msg)
					if err != nil {
						rr.l.Error("failed to send reminder", slog.Any("error", err))
					}
				})

				if err != nil {
					rr.l.Error("failed to schedule reminder", slog.Any("error", err))
				} else {
					rr.rMap[reminder.Id] = crId
					rr.l.Debug("reminder added", slog.Any("id", reminder.Id))
				}
			}

			for _, ci := range rr.crS {
				c.Remove(ci)
				rr.l.Debug("one of crones was removed", slog.Any("id", ci))
			}
			rr.crS = []cron.EntryID{}

			time.Sleep(30 * time.Second)
		}
	}()

	c.Start()
}

func (rr *RemindRepository) DeleteRemind(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}

	result, err := rr.collection.DeleteOne(ctx, filter)
	if err != nil {
		rr.l.Error("failed to delete reminder", slog.Any("error", err))
		return err
	}

	if result.DeletedCount == 0 {
		rr.l.Warn("no reminders found with the given id", slog.Any("error", mongo.ErrNoDocuments))
		return mongo.ErrNoDocuments
	}

	
	rr.crS = append(rr.crS, rr.rMap[id])
	delete(rr.rMap, id)

	rr.l.Debug("reminder successfully deleted from map and database", slog.Any("id", id))

	return nil
}

func (rr *RemindRepository) ListRemindByChat(ctx context.Context, id int64) ([]Remind, error) {
	filter := bson.M{"chat_id": id}

	var reminders []Remind
	cursor, err := rr.collection.Find(ctx, filter)
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
