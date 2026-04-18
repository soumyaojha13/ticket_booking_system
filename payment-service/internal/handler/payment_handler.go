package handler

import (
	"context"
	"log"
	"time"

	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/timer"
	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/usecase"
	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/worker"
	seatv1 "github.com/soumyaojha/ticket-booking-system/proto/seat/v1"
)

type PaymentHandler struct {
	seatv1.UnimplementedPaymentServiceServer
	usecase         usecase.PaymentUsecase
	inventoryClient seatv1.SeatServiceClient
	timers          *timer.Manager
	workerPool      *worker.WorkerPool
}

func NewPaymentHandler(u usecase.PaymentUsecase, inventoryClient seatv1.SeatServiceClient, timers *timer.Manager, workerPool *worker.WorkerPool) *PaymentHandler {
	return &PaymentHandler{
		usecase:         u,
		inventoryClient: inventoryClient,
		timers:          timers,
		workerPool:      workerPool,
	}
}

// StartPaymentTimer starts a 5-minute payment timer and locks the seat
func (h *PaymentHandler) StartPaymentTimer(ctx context.Context, req *seatv1.PaymentTimerRequest) (*seatv1.PaymentTimerResponse, error) {
	log.Printf("Starting payment timer for seat %s (user: %s)\n", req.SeatId, req.UserId)

	// First, lock the seat via inventory service with context timeout.
	seatCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	seatResp, err := h.inventoryClient.LockSeat(seatCtx, &seatv1.LockSeatRequest{
		SeatId:       req.SeatId,
		UserId:       req.UserId,
		LockExpiryMs: 5 * 60 * 1000, // 5 minutes in milliseconds
	})

	if err != nil {
		log.Printf("Error locking seat: %v\n", err)
		return &seatv1.PaymentTimerResponse{
			SeatId:  req.SeatId,
			Status:  "FAILED",
			Message: "Failed to lock seat: " + err.Error(),
		}, err
	}

	log.Printf("Seat locked successfully: %s (status: %s)\n", seatResp.GetSeatId(), seatResp.GetStatus())

	// Create payment record.
	payment, err := h.usecase.StartPaymentTimer(ctx, req.SeatId, req.UserId, req.Amount)
	if err != nil {
		// Release the seat if payment timer fails
		h.inventoryClient.ReleaseSeat(seatCtx, &seatv1.ReleaseSeatRequest{
			SeatId: req.SeatId,
			Reason: "PAYMENT_TIMER_FAILED",
		})
		return &seatv1.PaymentTimerResponse{
			SeatId:  req.SeatId,
			Status:  "FAILED",
			Message: err.Error(),
		}, err
	}

	// Start the 5-minute countdown in Payment Service (single source of truth for expiry).
	if h.timers != nil {
		h.timers.Start(req.SeatId, 5*time.Minute, func() {
			log.Printf("Payment window expired for seat %s (user: %s)\n", req.SeatId, req.UserId)
			_ = h.usecase.ExpirePayment(context.Background(), req.SeatId)

			releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer releaseCancel()
			_, releaseErr := h.inventoryClient.ReleaseSeat(releaseCtx, &seatv1.ReleaseSeatRequest{
				SeatId: req.SeatId,
				Reason: "TIMEOUT",
			})
			if releaseErr != nil {
				log.Printf("Error releasing seat %s on timeout: %v\n", req.SeatId, releaseErr)
			}
		})
	}

	return &seatv1.PaymentTimerResponse{
		SeatId:  req.SeatId,
		Status:  payment.Status,
		Message: "Payment timer started, seat locked for 5 minutes",
	}, nil
}

// ConfirmPayment confirms payment and books the seat
func (h *PaymentHandler) ConfirmPayment(ctx context.Context, req *seatv1.ConfirmPaymentRequest) (*seatv1.PaymentTimerResponse, error) {
	log.Printf("Confirming payment for seat %s (user: %s, transaction: %s)\n", req.SeatId, req.UserId, req.TransactionId)

	// Queue the confirmation work through a buffered channel to handle flash-sale spikes.
	if h.workerPool == nil {
		return &seatv1.PaymentTimerResponse{
			SeatId:  req.SeatId,
			Status:  "FAILED",
			Message: "Payment worker pool not available",
		}, nil
	}

	resultCh := make(chan error, 1)
	submitErr := h.workerPool.Submit(worker.Job{
		Name: "confirm_payment",
		Result: resultCh,
		Do: func(jobCtx context.Context) error {
			_, err := h.usecase.ConfirmPayment(jobCtx, req.SeatId, req.UserId, req.Amount, req.TransactionId)
			return err
		},
	})
	if submitErr != nil {
		return &seatv1.PaymentTimerResponse{
			SeatId:  req.SeatId,
			Status:  "FAILED",
			Message: submitErr.Error(),
		}, nil
	}

	select {
	case jobErr := <-resultCh:
		if jobErr != nil {
			return &seatv1.PaymentTimerResponse{
				SeatId:  req.SeatId,
				Status:  "FAILED",
				Message: jobErr.Error(),
			}, jobErr
		}
	case <-ctx.Done():
		return &seatv1.PaymentTimerResponse{
			SeatId:  req.SeatId,
			Status:  "FAILED",
			Message: "Payment confirmation timed out",
		}, ctx.Err()
	}

	// Cancel the expiry timer once payment is confirmed.
	if h.timers != nil {
		h.timers.Stop(req.SeatId)
	}

	// Book the seat via inventory service with context timeout
	bookCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := h.inventoryClient.BookSeat(bookCtx, &seatv1.BookSeatRequest{
		SeatId: req.SeatId,
		UserId: req.UserId,
	})

	if err != nil {
		log.Printf("Error booking seat: %v\n", err)
		return &seatv1.PaymentTimerResponse{
			SeatId:  req.SeatId,
			Status:  "FAILED",
			Message: "Failed to book seat: " + err.Error(),
		}, err
	}

	return &seatv1.PaymentTimerResponse{
		SeatId:  req.SeatId,
		Status:  "SUCCESS",
		Message: "Seat successfully booked after payment confirmation",
	}, nil
}
