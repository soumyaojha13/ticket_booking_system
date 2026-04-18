package grpc

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/soumyaojha/ticket-booking-system/internal/postgres"
	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/handler"
	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/repository"
	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/timer"
	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/usecase"
	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/worker"
	seatv1 "github.com/soumyaojha/ticket-booking-system/proto/seat/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// StartPaymentServer starts the Payment gRPC server
func StartPaymentServer(port string, workerPoolSize int) {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create inventory service client with context timeout
	inventoryClient, err := createInventoryClient()
	if err != nil {
		log.Fatalf("Failed to create inventory client: %v", err)
	}

	// Initialize payment service components
	db, err := postgres.Open(context.Background())
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Failed to close PostgreSQL connection: %v", err)
		}
	}()

	paymentRepo, err := repository.NewPostgresPaymentRepo(db)
	if err != nil {
		log.Fatalf("Failed to initialize payment repository: %v", err)
	}
	paymentUsecase := usecase.NewPaymentUsecase(paymentRepo)
	timerMgr := timer.NewManager()

	// Start worker pool for queued payment confirmations (flash-sale spikes).
	wp := worker.NewWorkerPool(workerPoolSize, 1000)
	wp.Start()
	defer wp.Stop()

	paymentHandler := handler.NewPaymentHandler(paymentUsecase, inventoryClient, timerMgr, wp)

	// Create gRPC server
	s := grpc.NewServer()
	seatv1.RegisterPaymentServiceServer(s, paymentHandler)

	log.Printf("Payment Service starting on :%s\n", port)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// createInventoryClient creates a gRPC client for the inventory service with timeout
func createInventoryClient() (seatv1.SeatServiceClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	addr := os.Getenv("INVENTORY_GRPC_ADDR")
	if addr == "" {
		addr = "localhost:50051"
	}

	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	return seatv1.NewSeatServiceClient(conn), nil
}
