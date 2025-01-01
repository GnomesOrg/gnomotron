package main

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
	"time"
)

var MONGO_URI = os.Getenv("GNOMOTRON_MONGO_URI")
var MONGO_DB = os.Getenv("GNOMOTRON_MONGO_DB")

type User struct {
	Id   int    `bson:"id"`
	Name string `bson:"name"`
}

func main() {
	clientOptions := options.Client().ApplyURI(MONGO_URI)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	user := User{7, "xxx"}

	res, _ := client.Database(MONGO_DB).Collection("users").InsertOne(ctx, user)
	log.Println(res.InsertedID)

	var usr User
	err = client.Database(MONGO_DB).Collection("users").FindOne(ctx, bson.D{{"name", "xxx"}}).Decode(&usr)
	log.Println(usr.Name)
}
