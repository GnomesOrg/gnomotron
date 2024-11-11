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

const collectionChats = "chats"

type Chat struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`
}

type Sender struct {
	ID   primitive.ObjectID `bson:"_id,omitempty"`
	Name string             `bson:"name"`
}

type Message struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
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

	collection := client.Database(cfg.MONGO_DB).Collection(collectionChats)
	return &ChatRepository{client: client, collection: collection}, nil
}

func (r *ChatRepository) AddMessage(chatID, senderID primitive.ObjectID, body string) {
	message := Message{
		ID:     primitive.NewObjectID(),
		Body:   body,
		Sender: senderID,
		ChatID: chatID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.collection.InsertOne(ctx, message)
	if err != nil {
		log.Printf("Can't push message to db: %+v", err)
	}
}
