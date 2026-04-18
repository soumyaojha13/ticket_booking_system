package usecase

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/model"
	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/repository"
)

const (
	PAYMENT_TIMEOUT = 5 * time.Minute
)

type PaymentUsecase interface {
	StartPaymentTimer(ctx context.Context, seatID, userID string, amount int32) (*model.Payment, error)
	ConfirmPayment(ctx context.Context, seatID, userID string, amount int32, transactionID string) (*model.Payment, error)
	GetPaymentStatus(ctx context.Context, seatID string) (*model.Payment, error)
	CancelPayment(ctx context.Context, seatID string) error
	ExpirePayment(ctx context.Context, seatID string) error
}

type paymentUsecase struct {
	repo repository.PaymentRepository
}

func NewPaymentUsecase(repo repository.PaymentRepository) PaymentUsecase {
	return &paymentUsecase{repo: repo}
}

// StartPaymentTimer creates a new payment timer for a seat
func (u *paymentUsecase) StartPaymentTimer(ctx context.Context, seatID, userID string, amount int32) (*model.Payment, error) {
	if seatID == "" || userID == "" {
		return nil, errors.New("seat_id and user_id are required")
	}

	// If there is an old payment record for this seat, allow reuse only when it is no longer active.
	if existing, err := u.repo.Get(seatID); err == nil {
		if existing.Status == "PENDING" && !existing.IsExpired() {
			return nil, errors.New("payment already exists for this seat")
		}
		if existing.Status == "SUCCESS" {
			return nil, errors.New("payment already exists for this seat")
		}

		_ = u.repo.Delete(seatID)
	} else if err != repository.ErrPaymentNotFound {
		return nil, err
	}

	now := time.Now()
	payment := &model.Payment{
		SeatID:    seatID,
		UserID:    userID,
		Amount:    amount,
		Status:    "PENDING",
		CreatedAt: now,
		ExpiresAt: now.Add(PAYMENT_TIMEOUT),
	}

	err := u.repo.Create(payment)
	if err != nil {
		return nil, err
	}

	log.Printf("Payment timer started for seat %s (user: %s), expires at %s\n", seatID, userID, payment.ExpiresAt)
	return payment, nil
}

// ConfirmPayment marks a payment as successful and confirms the booking
func (u *paymentUsecase) ConfirmPayment(ctx context.Context, seatID, userID string, amount int32, transactionID string) (*model.Payment, error) {
	if seatID == "" || userID == "" || transactionID == "" {
		return nil, errors.New("seat_id, user_id, and transaction_id are required")
	}

	payment, err := u.repo.Get(seatID)
	if err != nil {
		return nil, err
	}

	// Verify payment belongs to the user
	if payment.UserID != userID {
		return nil, errors.New("payment does not belong to this user")
	}

	if payment.Status != "PENDING" {
		return nil, errors.New("payment is not pending")
	}

	if time.Now().After(payment.ExpiresAt) {
		payment.Status = "EXPIRED"
		_ = u.repo.Update(payment)
		return nil, errors.New("payment expired")
	}

	// Update payment to success
	payment.Status = "SUCCESS"
	payment.TransactionID = transactionID

	err = u.repo.Update(payment)
	if err != nil {
		return nil, err
	}

	log.Printf("Payment confirmed for seat %s (user: %s, transaction: %s)\n", seatID, userID, transactionID)
	return payment, nil
}

// GetPaymentStatus retrieves the current payment status
func (u *paymentUsecase) GetPaymentStatus(ctx context.Context, seatID string) (*model.Payment, error) {
	return u.repo.Get(seatID)
}

// CancelPayment cancels a pending payment
func (u *paymentUsecase) CancelPayment(ctx context.Context, seatID string) error {
	payment, err := u.repo.Get(seatID)
	if err != nil {
		return err
	}

	if payment.Status != "PENDING" {
		return errors.New("can only cancel pending payments")
	}

	err = u.repo.Delete(seatID)
	if err != nil {
		return err
	}

	log.Printf("Payment cancelled for seat %s\n", seatID)
	return nil
}

func (u *paymentUsecase) ExpirePayment(ctx context.Context, seatID string) error {
	payment, err := u.repo.Get(seatID)
	if err != nil {
		return err
	}
	if payment.Status != "PENDING" {
		return nil
	}
	payment.Status = "EXPIRED"
	return u.repo.Update(payment)
}
