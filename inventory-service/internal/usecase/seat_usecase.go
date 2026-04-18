package usecase

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/soumyaojha/ticket-booking-system/inventory-service/internal/model"
	"github.com/soumyaojha/ticket-booking-system/inventory-service/internal/repository"
)

const (
	SEAT_LOCK_DURATION = 5 * time.Minute
)

type SeatUsecase interface {
	LockSeat(ctx context.Context, seatID, userID string) (*model.Seat, error)
	ReleaseSeat(ctx context.Context, seatID string, reason string) (*model.Seat, error)
	BookSeat(ctx context.Context, seatID, userID string) (*model.Seat, error)
	GetSeatStatus(ctx context.Context, seatID string) (*model.Seat, error)
}

type seatUsecase struct {
	repo repository.SeatRepository
}

func NewSeatUsecase(r repository.SeatRepository) SeatUsecase {
	return &seatUsecase{repo: r}
}

// LockSeat locks a seat for 5 minutes
func (u *seatUsecase) LockSeat(ctx context.Context, seatID, userID string) (*model.Seat, error) {
	if seatID == "" || userID == "" {
		return nil, errors.New("seat_id and user_id are required")
	}

	// Lock the seat with 5-minute expiry timestamp.
	// Countdown/expiry enforcement is owned by the Payment/Timer Service.
	seat, err := u.repo.Lock(seatID, userID, SEAT_LOCK_DURATION)
	if err != nil {
		return nil, err
	}

	log.Printf("Seat %s locked for user %s until %s\n", seatID, userID, seat.ExpiresAt)
	return seat, nil
}

// ReleaseSeat releases a locked seat
func (u *seatUsecase) ReleaseSeat(ctx context.Context, seatID string, reason string) (*model.Seat, error) {
	if seatID == "" {
		return nil, errors.New("seat_id is required")
	}


	seat, err := u.repo.Release(seatID, reason)
	if err != nil {
		return nil, err
	}

	log.Printf("Seat %s released. Reason: %s\n", seatID, reason)
	return seat, nil
}

// BookSeat confirms booking after payment
func (u *seatUsecase) BookSeat(ctx context.Context, seatID, userID string) (*model.Seat, error) {
	if seatID == "" || userID == "" {
		return nil, errors.New("seat_id and user_id are required")
	}

	seat, err := u.repo.Book(seatID, userID)
	if err != nil {
		return nil, err
	}

	log.Printf("Seat %s booked for user %s\n", seatID, userID)
	return seat, nil
}

// GetSeatStatus retrieves the current status of a seat
func (u *seatUsecase) GetSeatStatus(ctx context.Context, seatID string) (*model.Seat, error) {
	if seatID == "" {
		return nil, errors.New("seat_id is required")
	}

	seat, exists := u.repo.Get(seatID)
	if !exists {
		// Return available seat if it doesn't exist
		return &model.Seat{
			ID:     seatID,
			Status: "AVAILABLE",
		}, nil
	}

	return seat, nil
}
