//go:build ignore

package mongo

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	defaultURI      = "mongodb://mongo:27017"
	defaultDatabase = "ticket_booking"
	connectTimeout  = 10 * time.Second
)

func Connect(ctx context.Context) (*mongo.Client, string, error) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = defaultURI
	}

	dbName := os.Getenv("MONGO_DATABASE")
	if dbName == "" {
		dbName = defaultDatabase
	}

	connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	client, err := mongo.Connect(connectCtx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, "", fmt.Errorf("connect mongo: %w", err)
	}

	pingCtx, pingCancel := context.WithTimeout(ctx, connectTimeout)
	defer pingCancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, "", fmt.Errorf("ping mongo: %w", err)
	}

	return client, dbName, nil
}
