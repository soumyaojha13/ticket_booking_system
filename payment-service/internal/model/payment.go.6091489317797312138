package model

import (
	"time"
)

type Payment struct {
	SeatID        string    `bson:"seatId"`
	UserID        string    `bson:"userId"`
	Amount        int32     `bson:"amount"`
	Status        string    `bson:"status"` // PENDING, SUCCESS, EXPIRED, CANCELLED
	CreatedAt     time.Time `bson:"createdAt"`
	ExpiresAt     time.Time `bson:"expiresAt"`
	TransactionID string    `bson:"transactionId"`
}

// IsExpired checks if the payment timer has expired
func (p *Payment) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}
