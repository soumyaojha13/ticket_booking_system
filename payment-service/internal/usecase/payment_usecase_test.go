package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/model"
	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/repository"
)

func TestPaymentUsecaseStartAndConfirmPayment(t *testing.T) {
	t.Parallel()

	uc := NewPaymentUsecase(repository.NewPaymentRepo())
	ctx := context.Background()

	payment, err := uc.StartPaymentTimer(ctx, "A1", "user-1", 1000)
	if err != nil {
		t.Fatalf("StartPaymentTimer returned error: %v", err)
	}

	if payment.Status != "PENDING" {
		t.Fatalf("expected status PENDING, got %s", payment.Status)
	}

	confirmed, err := uc.ConfirmPayment(ctx, "A1", "user-1", 1000, "txn-1")
	if err != nil {
		t.Fatalf("ConfirmPayment returned error: %v", err)
	}

	if confirmed.Status != "SUCCESS" {
		t.Fatalf("expected status SUCCESS, got %s", confirmed.Status)
	}

	if confirmed.TransactionID != "txn-1" {
		t.Fatalf("expected transaction ID txn-1, got %s", confirmed.TransactionID)
	}
}

func TestPaymentUsecaseRejectsDifferentUser(t *testing.T) {
	t.Parallel()

	uc := NewPaymentUsecase(repository.NewPaymentRepo())
	ctx := context.Background()

	if _, err := uc.StartPaymentTimer(ctx, "A2", "user-1", 1000); err != nil {
		t.Fatalf("StartPaymentTimer returned error: %v", err)
	}

	_, err := uc.ConfirmPayment(ctx, "A2", "user-2", 1000, "txn-2")
	if err == nil || !strings.Contains(err.Error(), "does not belong") {
		t.Fatalf("expected ownership error, got %v", err)
	}
}

func TestPaymentUsecaseRejectsExpiredPayment(t *testing.T) {
	t.Parallel()

	repo := repository.NewPaymentRepo()
	uc := NewPaymentUsecase(repo)
	ctx := context.Background()

	err := repo.Create(&model.Payment{
		SeatID:    "A3",
		UserID:    "user-1",
		Amount:    1000,
		Status:    "PENDING",
		CreatedAt: time.Now().Add(-10 * time.Minute),
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	_, err = uc.ConfirmPayment(ctx, "A3", "user-1", 1000, "txn-3")
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired error, got %v", err)
	}

	payment, err := repo.Get("A3")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if payment.Status != "EXPIRED" {
		t.Fatalf("expected stored payment status EXPIRED, got %s", payment.Status)
	}
}

func TestPaymentUsecaseCancelPayment(t *testing.T) {
	t.Parallel()

	repo := repository.NewPaymentRepo()
	uc := NewPaymentUsecase(repo)
	ctx := context.Background()

	if _, err := uc.StartPaymentTimer(ctx, "A4", "user-1", 1000); err != nil {
		t.Fatalf("StartPaymentTimer returned error: %v", err)
	}

	if err := uc.CancelPayment(ctx, "A4"); err != nil {
		t.Fatalf("CancelPayment returned error: %v", err)
	}

	if _, err := repo.Get("A4"); err == nil {
		t.Fatal("expected cancelled payment to be removed from repository")
	}
}

