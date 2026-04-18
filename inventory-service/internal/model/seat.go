package model

import (
	"time"
)

type Seat struct {
	ID        string    `bson:"id"`
	Status    string    `bson:"status"`    // AVAILABLE, LOCKED, BOOKED
	UserID    string    `bson:"userId"`    // User that currently owns the lock/booking
	LockedAt  time.Time `bson:"lockedAt"`  // When the seat was locked
	ExpiresAt time.Time `bson:"expiresAt"` // When the lock expires (5 minutes after locked)
}

// IsExpired checks if the seat lock has expired
func (s *Seat) IsExpired() bool {
	if s.Status != "LOCKED" {
		return false
	}
	return time.Now().After(s.ExpiresAt)
}

// GetTimeUntilExpiry returns the duration until the seat expires
func (s *Seat) GetTimeUntilExpiry() time.Duration {
	if s.Status != "LOCKED" {
		return 0
	}
	remaining := time.Until(s.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}
