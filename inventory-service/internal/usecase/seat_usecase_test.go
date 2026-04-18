package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/soumyaojha/ticket-booking-system/inventory-service/internal/repository"
)

func TestSeatUsecaseLockBookAndRelease(t *testing.T) {
	t.Parallel()

	uc := NewSeatUsecase(repository.NewSeatRepo())
	ctx := context.Background()

	seat, err := uc.LockSeat(ctx, "A1", "user-1")
	if err != nil {
		t.Fatalf("LockSeat returned error: %v", err)
	}

	if seat.Status != "LOCKED" {
		t.Fatalf("expected seat to be LOCKED, got %s", seat.Status)
	}

	if seat.UserID != "user-1" {
		t.Fatalf("expected seat user to be user-1, got %s", seat.UserID)
	}

	if remaining := time.Until(seat.ExpiresAt); remaining < 4*time.Minute || remaining > 5*time.Minute+5*time.Second {
		t.Fatalf("expected expiry around 5 minutes, got %s", remaining)
	}

	booked, err := uc.BookSeat(ctx, "A1", "user-1")
	if err != nil {
		t.Fatalf("BookSeat returned error: %v", err)
	}

	if booked.Status != "BOOKED" {
		t.Fatalf("expected seat to be BOOKED, got %s", booked.Status)
	}

	released, err := uc.ReleaseSeat(ctx, "A1", "MANUAL_CANCEL")
	if err != nil {
		t.Fatalf("ReleaseSeat returned error: %v", err)
	}

	if released.Status != "AVAILABLE" {
		t.Fatalf("expected seat to be AVAILABLE after release, got %s", released.Status)
	}
}

func TestSeatUsecaseRejectsDoubleLock(t *testing.T) {
	t.Parallel()

	uc := NewSeatUsecase(repository.NewSeatRepo())
	ctx := context.Background()

	if _, err := uc.LockSeat(ctx, "A2", "user-1"); err != nil {
		t.Fatalf("first LockSeat returned error: %v", err)
	}

	_, err := uc.LockSeat(ctx, "A2", "user-2")
	if !errors.Is(err, repository.ErrSeatAlreadyLocked) {
		t.Fatalf("expected ErrSeatAlreadyLocked, got %v", err)
	}
}

func TestSeatUsecaseReturnsAvailableForUnknownSeat(t *testing.T) {
	t.Parallel()

	uc := NewSeatUsecase(repository.NewSeatRepo())

	seat, err := uc.GetSeatStatus(context.Background(), "Z9")
	if err != nil {
		t.Fatalf("GetSeatStatus returned error: %v", err)
	}

	if seat.Status != "AVAILABLE" {
		t.Fatalf("expected unknown seat to be AVAILABLE, got %s", seat.Status)
	}
}

