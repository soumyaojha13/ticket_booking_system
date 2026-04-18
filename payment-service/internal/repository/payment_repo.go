package repository

import (
	"errors"
	"sync"

	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/model"
)

var (
	ErrPaymentNotFound = errors.New("payment not found")
)

type PaymentRepository interface {
	Create(payment *model.Payment) error
	Get(seatID string) (*model.Payment, error)
	Update(payment *model.Payment) error
	Delete(seatID string) error
	GetAll() []*model.Payment
}

type inMemoryPaymentRepo struct {
	data map[string]*model.Payment
	mu   sync.RWMutex
}

func NewPaymentRepo() PaymentRepository {
	return &inMemoryPaymentRepo{
		data: make(map[string]*model.Payment),
	}
}

func (r *inMemoryPaymentRepo) Create(payment *model.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[payment.SeatID]; exists {
		return errors.New("payment already exists for this seat")
	}

	r.data[payment.SeatID] = payment
	return nil
}

func (r *inMemoryPaymentRepo) Get(seatID string) (*model.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	payment, exists := r.data[seatID]
	if !exists {
		return nil, ErrPaymentNotFound
	}

	return payment, nil
}

func (r *inMemoryPaymentRepo) Update(payment *model.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[payment.SeatID]; !exists {
		return ErrPaymentNotFound
	}

	r.data[payment.SeatID] = payment
	return nil
}

func (r *inMemoryPaymentRepo) Delete(seatID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[seatID]; !exists {
		return ErrPaymentNotFound
	}

	delete(r.data, seatID)
	return nil
}

func (r *inMemoryPaymentRepo) GetAll() []*model.Payment {
	r.mu.RLock()
	defer r.mu.RUnlock()

	payments := make([]*model.Payment, 0, len(r.data))
	for _, payment := range r.data {
		if !payment.IsExpired() {
			payments = append(payments, payment)
		}
	}
	return payments
}
