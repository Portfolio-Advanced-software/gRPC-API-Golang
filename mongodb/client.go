package mongodb

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConnectToMongoDB creates a new MongoDB client and returns a pointer to the client
func ConnectToMongoDB(connUri string) *mongo.Client {
	// non-nil empty context
	mongoCtx := context.Background()
	// Connect takes in a context and options, the connection URI is the only option we pass for now
	client, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(connUri))
	// Handle potential errors
	if err != nil {
		log.Fatal(err)
	}

	// Check whether the connection was successful by pinging the MongoDB server
	err = client.Ping(mongoCtx, nil)
	if err != nil {
		log.Fatalf("Could not connect to MongoDB: %v\n", err)
	} else {
		fmt.Println("Connected to MongoDB")
	}

	return client
}
