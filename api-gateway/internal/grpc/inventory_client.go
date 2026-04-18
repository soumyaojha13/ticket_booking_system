package grpc

import (
	"context"
	"os"
	"time"

	seatv1 "github.com/soumyaojha/ticket-booking-system/proto/seat/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type InventoryClient struct {
	client seatv1.SeatServiceClient
	conn   *grpc.ClientConn
}

func NewInventoryClient() (*InventoryClient, error) {
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

	return &InventoryClient{
		client: seatv1.NewSeatServiceClient(conn),
		conn:   conn,
	}, nil
}

// LockSeat locks a seat with context timeout (5 seconds)
func (c *InventoryClient) LockSeat(ctx context.Context, seatID string, userID string) (*seatv1.SeatResponse, error) {
	// Create a timeout context for the gRPC call
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.LockSeat(callCtx, &seatv1.LockSeatRequest{
		SeatId:       seatID,
		UserId:       userID,
		LockExpiryMs: 5 * 60 * 1000, // 5 minutes in milliseconds
	})
	return resp, err
}

// GetSeatStatus retrieves the current seat status with timeout
func (c *InventoryClient) GetSeatStatus(ctx context.Context, seatID string) (*seatv1.SeatResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.GetSeatStatus(callCtx, &seatv1.SeatStatusRequest{
		SeatId: seatID,
	})
	return resp, err
}

// ReleaseSeat releases a locked seat with timeout
func (c *InventoryClient) ReleaseSeat(ctx context.Context, seatID string, reason string) (*seatv1.SeatResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.ReleaseSeat(callCtx, &seatv1.ReleaseSeatRequest{
		SeatId: seatID,
		Reason: reason,
	})
	return resp, err
}

// BookSeat confirms booking with timeout
func (c *InventoryClient) BookSeat(ctx context.Context, seatID string, userID string) (*seatv1.SeatResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.BookSeat(callCtx, &seatv1.BookSeatRequest{
		SeatId: seatID,
		UserId: userID,
	})
	return resp, err
}

// SeatStream opens a bi-directional stream to receive seat events.
func (c *InventoryClient) SeatStream(ctx context.Context) (seatv1.SeatService_SeatStreamClient, error) {
	return c.client.SeatStream(ctx)
}

func (c *InventoryClient) Close() error {
	return c.conn.Close()
}
