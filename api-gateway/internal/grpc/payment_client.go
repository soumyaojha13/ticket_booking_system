package grpc

import (
	"context"
	"os"
	"time"

	seatv1 "github.com/soumyaojha/ticket-booking-system/proto/seat/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PaymentClient struct {
	client seatv1.PaymentServiceClient
	conn   *grpc.ClientConn
}

func NewPaymentClient() (*PaymentClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	addr := os.Getenv("PAYMENT_GRPC_ADDR")
	if addr == "" {
		addr = "localhost:50052"
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

	return &PaymentClient{
		client: seatv1.NewPaymentServiceClient(conn),
		conn:   conn,
	}, nil
}

// StartPaymentTimer starts a payment timer with context timeout
func (c *PaymentClient) StartPaymentTimer(ctx context.Context, seatID string, userID string, amount int32) (*seatv1.PaymentTimerResponse, error) {
	// Create a timeout context for the gRPC call
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := c.client.StartPaymentTimer(callCtx, &seatv1.PaymentTimerRequest{
		SeatId: seatID,
		UserId: userID,
		Amount: amount,
	})
	return resp, err
}

// ConfirmPayment confirms payment and books the seat
func (c *PaymentClient) ConfirmPayment(ctx context.Context, seatID string, userID string, amount int32, transactionID string) (*seatv1.PaymentTimerResponse, error) {
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := c.client.ConfirmPayment(callCtx, &seatv1.ConfirmPaymentRequest{
		SeatId:        seatID,
		UserId:        userID,
		Amount:        amount,
		TransactionId: transactionID,
	})
	return resp, err
}

// ProcessPayment simulates payment processing
func (c *PaymentClient) ProcessPayment(seatID string) (string, error) {
	// For now, return pending status
	return "PENDING", nil
}

func (c *PaymentClient) Close() error {
	return c.conn.Close()
}
