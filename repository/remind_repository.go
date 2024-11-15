package repository

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"time"
)

const remindCollection = "remind"

type RemindRepository struct {
	client     *mongo.Client
	collection *mongo.Collection
}

type Remind struct {
	Id      primitive.ObjectID `bson:"_id,omitempty"`
	Cron    string             `bson:"cron"`
	Message string             `bson:"message"`
	ChatID  string             `bson:"chat_id"`
}

func NewRemindRepository(client *mongo.Client, collection *mongo.Collection) *RemindRepository {
	return &RemindRepository{client, collection}
}

func (rRepo *RemindRepository) addRemind(r Remind) *mongo.InsertOneResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rUUID, err := rRepo.collection.InsertOne(ctx, r)
	if err != nil {
		log.Printf("Can't push reminder to db: %+v", err)
	}

	return rUUID
}
