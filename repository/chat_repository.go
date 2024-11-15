package repository

import (
	"context"
	"flabergnomebot/config"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const chatsCollection = "chats"

type Chat struct {
	ID primitive.ObjectID `bson:"_id"`
}

type Sender struct {
	ID   primitive.ObjectID `bson:"_id"`
	Name string             `bson:"name"`
}

type Message struct {
	ID     primitive.ObjectID `bson:"_id"`
	Body   string             `bson:"body"`
	Sender primitive.ObjectID `bson:"sender_id"`
	ChatID primitive.ObjectID `bson:"chat_id"`
}

type ChatRepository struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func NewChatRepository(cfg *config.Config) (*ChatRepository, error) {
	clientOptions := options.Client().ApplyURI(cfg.MONGO_URI)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatalf("Error on db connection: %+v", err)
	}

	collection := client.Database(cfg.MONGO_DB).Collection(chatsCollection)
	return &ChatRepository{client: client, collection: collection}, nil
}

func (r *ChatRepository) AddMessage(chatID, senderID primitive.ObjectID, body string) {
	m := Message{
		ID:     primitive.NewObjectID(),
		Body:   body,
		Sender: senderID,
		ChatID: chatID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.collection.InsertOne(ctx, m)
	if err != nil {
		log.Printf("Can't push m to db: %+v", err)
	}
}
