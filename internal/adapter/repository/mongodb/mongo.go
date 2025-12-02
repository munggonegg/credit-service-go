package mongodb

import (
	"context"
	"log"
	"time"

	"munggonegg/credit-service-go/internal/config"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client
var DB *mongo.Database

func Connect() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(config.AppConfig.MongoURL)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	Client = client
	DB = client.Database(config.AppConfig.MongoDBName)
	log.Println("Connected to MongoDB")

	EnsureIndexes()
}

func EnsureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	createIndex(ctx, config.UsageEventColl, bson.D{{Key: "userId", Value: 1}}, false)
	createIndex(ctx, config.UserBalanceColl, bson.D{{Key: "userId", Value: 1}}, true)
	createIndex(ctx, config.UserMainPackageColl, bson.D{{Key: "userId", Value: 1}}, true)
	createIndex(ctx, config.UserTopupPackageColl, bson.D{{Key: "userId", Value: 1}}, true)
	createIndex(ctx, config.TopupPackageEventColl, bson.D{{Key: "topupId", Value: 1}}, true)
	createIndex(ctx, config.SubsPackageEventColl, bson.D{{Key: "subscriptionEventId", Value: 1}}, true)
}

func createIndex(ctx context.Context, collectionName string, keys bson.D, unique bool) {
	collection := DB.Collection(collectionName)
	indexModel := mongo.IndexModel{
		Keys:    keys,
		Options: options.Index().SetUnique(unique),
	}
	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Printf("Index creation warning on %s: %v", collectionName, err)
	}
}

func GetCollection(name string) *mongo.Collection {
	return DB.Collection(name)
}
