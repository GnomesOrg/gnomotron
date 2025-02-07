package service

import (
	"context"
	"flabergnomebot/internal/config"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const MessageCollection = "message"

type Message struct {
	Id      primitive.ObjectID `bson:"_id,omitempty"`
	TelId   int                `bson:"telegram_id"`
	Body    string             `bson:"message"`
	ChatID  int64              `bson:"chat_id"`
	Replies []Message          `bson:"replies"`
	Uname   string             `bson:"username"`
}

func NewMessage(tId int, body string, chatId int64, replies []Message, uname string) *Message {
	return &Message{
		Id:      primitive.NewObjectID(),
		TelId:   tId,
		Body:    body,
		ChatID:  chatId,
		Replies: replies,
		Uname:   uname,
	}
}

type MessageRepository struct {
	c   *mongo.Collection
	l   *slog.Logger
	cfg *config.Config
}

func NewMessageRepository(c *mongo.Collection, l *slog.Logger, cfg *config.Config) *MessageRepository {
	return &MessageRepository{
		c:   c,
		l:   l,
		cfg: cfg,
	}
}

func (mr *MessageRepository) FindMessageByTelegramId(ctx context.Context, tId int) (*Message, error) {
	f := bson.D{{Key: "telegram_id", Value: tId}}

	cur, err := mr.c.Find(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("FindMessageByTelegramIderror %w", err)
	}
	defer cur.Close(ctx)

	var m Message
	for cur.Next(ctx) {
		if curErr := cur.Decode(&m); curErr != nil {
			return nil, fmt.Errorf("CursorError %w", err)
		}
	}

	return &m, nil
}

func (mr *MessageRepository) AddMessage(ctx context.Context, m Message) (*mongo.InsertOneResult, error) {
	maxDs := mr.cfg.MAX_DIALOGUE_SIZE

	if len(m.Replies) > int(maxDs) {
		m.Replies = m.Replies[len(m.Replies)-int(maxDs):]
	}

	mUUID, err := mr.c.InsertOne(ctx, m)
	mr.l.Debug("new dialogue message", slog.String("message body: ", m.Body))

	if err != nil {
		return nil, err
	}

	return mUUID, nil
}
