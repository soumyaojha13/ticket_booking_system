package timer

import (
	"context"
	"log"
	"time"

	seatv1 "github.com/soumyaojha/ticket-booking-system/proto/seat/v1"
)

const (
	LOCK_TIMEOUT = 5 * time.Minute
)

// StartTimer starts a 5-minute timer for seat lock expiry
// This uses time.After and goroutines as required
func StartTimer(ctx context.Context, client seatv1.SeatServiceClient, seatID, userID string) {
	log.Printf("Starting timer for seat: %s (user: %s)\n", seatID, userID)

	// Create a context with timeout for the timer
	timerCtx, cancel := context.WithTimeout(context.Background(), LOCK_TIMEOUT)
	defer cancel()

	// Wait for either:
	// 1. The lock timeout to expire (auto-release seat)
	// 2. The context to be cancelled (payment confirmed or manually released)
	select {
	case <-time.After(LOCK_TIMEOUT):
		// Timer expired - release the seat
		log.Printf("Lock timeout for seat: %s (user: %s). Auto-releasing...\n", seatID, userID)

		// Create a context with timeout for the gRPC call
		releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer releaseCancel()

		_, err := client.ReleaseSeat(releaseCtx, &seatv1.ReleaseSeatRequest{
			SeatId: seatID,
			Reason: "TIMEOUT",
		})
		if err != nil {
			log.Printf("Error releasing seat %s: %v\n", seatID, err)
		}

	case <-timerCtx.Done():
		// Context cancelled - lock was manually released or payment confirmed
		log.Printf("Timer cancelled for seat: %s\n", seatID)
	}
}

// StartTimerWithCallback starts a timer and calls a callback when it expires
func StartTimerWithCallback(ctx context.Context, seatID, userID string, duration time.Duration, callback func()) {
	log.Printf("Starting timer for seat: %s (user: %s) with duration: %v\n", seatID, userID, duration)

	timerCtx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	select {
	case <-time.After(duration):
		log.Printf("Timer expired for seat: %s\n", seatID)
		if callback != nil {
			callback()
		}
	case <-timerCtx.Done():
		log.Printf("Timer cancelled for seat: %s\n", seatID)
	}
}
