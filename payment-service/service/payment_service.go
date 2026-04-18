// internal/service/payment_service.go
package service

import (
	"time"
)

type PaymentService struct{}

func NewPaymentService() *PaymentService {
	return &PaymentService{}
}

func (s *PaymentService) Process(seatID string) string {
	// simulate payment processing
	time.Sleep(10 * time.Second)

	// simulate failure (you can randomize)
	return "FAILED" // or "SUCCESS"
}
