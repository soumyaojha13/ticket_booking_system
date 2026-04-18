package grpc

import (
	"context"
	"log"
	"time"

	seatv1 "github.com/soumyaojha/ticket-booking-system/proto/seat/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewInventoryClient creates a new gRPC client for the inventory service
// with proper error handling
func NewInventoryClient() seatv1.SeatServiceClient {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Failed to connect to inventory service: %v", err)
	}

	return seatv1.NewSeatServiceClient(conn)
}

// NotifyInventory sends a notification to the inventory service about payment status
// Demonstrates context.WithTimeout for gRPC calls
func NotifyInventory(ctx context.Context, client seatv1.SeatServiceClient, seatID string, status string) error {
	// Create a timeout context for the gRPC call
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if status == "FAILED" || status == "TIMEOUT" {
		_, err := client.ReleaseSeat(callCtx, &seatv1.ReleaseSeatRequest{
			SeatId: seatID,
			Reason: status,
		})
		return err
	} else if status == "SUCCESS" {
		// This would be called from ConfirmPayment in handler
		log.Printf("Payment successful for seat %s\n", seatID)
	}

	return nil
}
