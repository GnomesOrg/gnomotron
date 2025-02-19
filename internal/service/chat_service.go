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
const ChatCollection = "chat"

type Message struct {
	Id      primitive.ObjectID `bson:"_id,omitempty"`
	TelId   int                `bson:"telegram_id"`
	Body    string             `bson:"message"`
	ChatID  int64              `bson:"chat_id"`
	Replies []Message          `bson:"replies"`
	Uname   string             `bson:"username"`
}

type Chat struct {
	Id     primitive.ObjectID `bson:"_id,omitempty"`
	ChatID int64              `bson:"chatId"`
	Name   string             `bson:"name"`
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

func NewChat(chatId int64, name string) *Chat {
	return &Chat{
		Id:     primitive.NewObjectID(),
		ChatID: chatId,
		Name:   name,
	}
}

type Repositroy struct {
	c     *mongo.Collection
	l     *slog.Logger
	cfg   *config.Config
}

func NewRepository(c *mongo.Collection, l *slog.Logger, cfg *config.Config) *Repositroy {
	return &Repositroy{
		c:   c,
		l:   l,
		cfg: cfg,
	}
}

func (r *Repositroy) FindMessageByTelegramId(ctx context.Context, tId int) (*Message, error) {
	f := bson.D{{Key: "telegram_id", Value: tId}}

	cur, err := r.c.Find(ctx, f)
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

func (r *Repositroy) AddMessage(ctx context.Context, m Message) error {
	maxDs := r.cfg.MAX_DIALOGUE_SIZE

	if len(m.Replies) > int(maxDs) {
		m.Replies = m.Replies[len(m.Replies)-int(maxDs):]
	}

	_, err := r.c.InsertOne(ctx, m)
	r.l.Debug("new dialogue message", slog.String("message body: ", m.Body))

	if err != nil {
		return err
	}

	return nil
}

func (r *Repositroy) AddChat(ctx context.Context, c Chat) error {
	filter := bson.M{"chatId": c.ChatID}
	var existingChat Chat
	err := r.c.FindOne(ctx, filter).Decode(&existingChat)

	if err != mongo.ErrNoDocuments {
		return fmt.Errorf("failed to check for existing chat: %w", err)
	}

	_, err = r.c.InsertOne(ctx, c)
	if err != nil {
		return fmt.Errorf("failed to insert chat: %w", err)
	}

	r.l.Debug("new chat registered", slog.String("chat name: ", c.Name), slog.Int64("chatId", c.ChatID))

	return nil
}
