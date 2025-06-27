package service

import (
	"context"
	"errors"
	"flabergnomebot/internal/config"
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
	Id               primitive.ObjectID `bson:"_id,omitempty"`
	ChatID           int64              `bson:"chatId"`
	Name             string             `bson:"name"`
	ReplyProbability float32            `bson:"reply_probability"`
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
		Id:               primitive.NewObjectID(),
		ChatID:           chatId,
		Name:             name,
		ReplyProbability: 0,
	}
}

type ChatRepository struct {
	cCol *mongo.Collection
	mCol *mongo.Collection
	l    *slog.Logger
	cfg  *config.Config
}

// Single responsibility has been violated. It's over...
func NewChatRepository(db *mongo.Database, l *slog.Logger, cfg *config.Config) *ChatRepository {
	return &ChatRepository{
		cCol: db.Collection(ChatCollection),
		mCol: db.Collection(MessageCollection),
		l:    l,
		cfg:  cfg,
	}
}

func (r *ChatRepository) FindMessageByTelegramId(ctx context.Context, tId int) (*Message, error) {
	f := bson.D{{Key: "telegram_id", Value: tId}}

	cur, err := r.mCol.Find(ctx, f)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var m Message
	for cur.Next(ctx) {
		if curErr := cur.Decode(&m); curErr != nil {
			return nil, err
		}
	}

	return &m, nil
}

func (r *ChatRepository) AddMessage(ctx context.Context, m Message) error {
	maxDs := r.cfg.MAX_DIALOGUE_SIZE

	if len(m.Replies) > int(maxDs) {
		m.Replies = m.Replies[len(m.Replies)-int(maxDs):]
	}

	_, err := r.mCol.InsertOne(ctx, m)
	r.l.Debug("new dialogue message", slog.String("message body: ", m.Body))

	if err != nil {
		return err
	}

	return nil
}

func (r *ChatRepository) AddChat(ctx context.Context, c Chat) error {
	filter := bson.M{"chatId": c.ChatID}
	var existingChat Chat
	err := r.cCol.FindOne(ctx, filter).Decode(&existingChat)

	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}

	if err == nil {
		return nil
	}

	_, err = r.cCol.InsertOne(ctx, c)
	if err != nil {
		return err
	}

	r.l.Debug("new chat registered", slog.String("chat name", c.Name), slog.Int64("chatId", c.ChatID))

	return nil
}


func (r *ChatRepository) FindChatByChatId(ctx context.Context, chatId int64) (*Chat, error) {
	f := bson.D{{Key: "chatId", Value: chatId}}

	cur, err := r.cCol.Find(ctx, f)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var c Chat
	for cur.Next(ctx) {
		if curErr := cur.Decode(&c); curErr != nil {
			return nil, err
		}
	}

	return &c, nil
}

func (r *ChatRepository) UpdateChat(ctx context.Context, chat *Chat) error {
	filter := bson.M{"chatId": chat.ChatID}
	update := bson.M{"$set": bson.M{"reply_probability": chat.ReplyProbability}}

	_, err := r.cCol.UpdateOne(ctx, filter, update)
	return err
}

func (r *ChatRepository) FindAllChats(ctx context.Context) ([]Chat, error) {
	cursor, err := r.cCol.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var chats []Chat
	if err := cursor.All(ctx, &chats); err != nil {
		return nil, err
	}

	return chats, nil
}
