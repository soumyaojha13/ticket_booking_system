package main

import (
	"context"
	"log"
	"time"

	"github.com/soumyaojha/ticket-booking-system/internal/postgres"
	"github.com/soumyaojha/ticket-booking-system/inventory-service/internal/cache"
	"github.com/soumyaojha/ticket-booking-system/inventory-service/internal/grpc"
	"github.com/soumyaojha/ticket-booking-system/inventory-service/internal/handler"
	"github.com/soumyaojha/ticket-booking-system/inventory-service/internal/repository"
	"github.com/soumyaojha/ticket-booking-system/inventory-service/internal/usecase"
)

func main() {
	redisClient := cache.NewRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}

	db, err := postgres.Open(context.Background())
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close PostgreSQL connection: %v", err)
		}
	}()

	repo, err := repository.NewHybridSeatRepo(repository.NewRedisSeatRepo(redisClient), db)
	if err != nil {
		log.Fatalf("failed to initialize seat repository: %v", err)
	}
	usecase := usecase.NewSeatUsecase(repo)
	handler := handler.NewSeatHandler(usecase)

	grpc.StartServer(handler)
}
