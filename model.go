package main

import (
	"context"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Message struct {
	Role    string `json:"role" bson:"role"`
	Content string `json:"content" bson:"content"`
	UserID  int64  `json:"-" bson:"user_id"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int      `json:"created"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Choice struct {
	Index        int    `json:"index"`
	FinishReason string `json:"finish_reason"`
	Message      Message
}

var client *mongo.Client

func InitDB() {
	// set up mongodb
	uri := os.Getenv("MONGODB_URI")
	log.Println(uri)
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)
	var err error
	client, err = mongo.Connect(context.TODO(), opts)
	if err != nil {
		panic(err)
	}
	// defer func() {
	// 	if err = client.Disconnect(context.TODO()); err != nil {
	// 		panic(err)
	// 	}
	// }()

	//check if document exists by user_id
	// collection := client.Database("go_chatgpt_telegram").Collection("messages")
}

func getMessagesByUserID(userID int64) []Message {
	collection := client.Database("go_chatgpt_telegram").Collection("messages")
	filter := bson.D{{"user_id", userID}}
	var messages []Message
	cur, err := collection.Find(context.TODO(), filter)
	if err != nil {
		log.Fatal(err)
	}
	for cur.Next(context.TODO()) {
		var message Message
		err := cur.Decode(&message)
		if err != nil {
			log.Fatal(err)
		}
		messages = append(messages, message)
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	cur.Close(context.TODO())
	return messages
}

func writeMessageToDB(message Message) error {

	collection := client.Database("go_chatgpt_telegram").Collection("messages")
	_, err := collection.InsertOne(context.TODO(), message)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func removeMessagesByUserID(userID int64) error {
	collection := client.Database("go_chatgpt_telegram").Collection("messages")
	filter := bson.D{{"user_id", userID}}
	_, err := collection.DeleteMany(context.TODO(), filter)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}
