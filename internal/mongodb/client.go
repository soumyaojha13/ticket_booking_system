//go:build ignore

package mongodb

import (
	"context"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	defaultURI      = "mongodb://localhost:27017"
	defaultDatabase = "ticket_booking"
)

func Enabled() bool {
	return os.Getenv("MONGODB_URI") != ""
}

func DatabaseName() string {
	if name := os.Getenv("MONGODB_DATABASE"); name != "" {
		return name
	}
	return defaultDatabase
}

func Connect(ctx context.Context) (*mongo.Client, *mongo.Database, error) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = defaultURI
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, nil, err
	}

	return client, client.Database(DatabaseName()), nil
}
