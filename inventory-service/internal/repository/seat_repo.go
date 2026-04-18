package repository

import (
	"sync"
	"time"

	"github.com/soumyaojha/ticket-booking-system/inventory-service/internal/model"
)

type SeatRepository interface {
	Get(seatID string) (*model.Seat, bool)
	Save(seat *model.Seat)
	Lock(seatID, userID string, lockDuration time.Duration) (*model.Seat, error)
	Release(seatID string, reason string) (*model.Seat, error)
	Book(seatID, userID string) (*model.Seat, error)
	GetAll() []*model.Seat
}

type InMemoryRepo struct {
	data map[string]*model.Seat
	mu   sync.RWMutex
}

func NewSeatRepo() SeatRepository {
	return &InMemoryRepo{
		data: make(map[string]*model.Seat),
	}
}

func (r *InMemoryRepo) Get(seatID string) (*model.Seat, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seat, ok := r.data[seatID]

	// Check if seat lock has expired
	if ok && seat.IsExpired() {
		return &model.Seat{
			ID:     seat.ID,
			Status: "AVAILABLE",
			UserID: "",
		}, true
	}

	return seat, ok
}

func (r *InMemoryRepo) Save(seat *model.Seat) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[seat.ID] = seat
}

func (r *InMemoryRepo) Lock(seatID, userID string, lockDuration time.Duration) (*model.Seat, error) {
	r.mu.Lock()

	// Check if seat exists and is available
	seat, exists := r.data[seatID]

	if exists && seat.Status == "LOCKED" && !seat.IsExpired() {
		r.mu.Unlock()
		return nil, ErrSeatAlreadyLocked
	}

	// Lock the seat
	now := time.Now()
	lockedSeat := &model.Seat{
		ID:        seatID,
		Status:    "LOCKED",
		UserID:    userID,
		LockedAt:  now,
		ExpiresAt: now.Add(lockDuration),
	}

	r.data[seatID] = lockedSeat
	r.mu.Unlock()

	return lockedSeat, nil
}

func (r *InMemoryRepo) Release(seatID string, reason string) (*model.Seat, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	seat, exists := r.data[seatID]

	if !exists {
		return nil, ErrSeatNotFound
	}

	seat.Status = "AVAILABLE"
	seat.UserID = ""
	r.data[seatID] = seat

	return seat, nil
}

func (r *InMemoryRepo) Book(seatID, userID string) (*model.Seat, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	seat, exists := r.data[seatID]

	if !exists {
		return nil, ErrSeatNotFound
	}

	if seat.Status == "LOCKED" && seat.UserID != userID {
		return nil, ErrSeatLockedByOtherUser
	}

	seat.Status = "BOOKED"
	seat.UserID = userID
	r.data[seatID] = seat

	return seat, nil
}

func (r *InMemoryRepo) GetAll() []*model.Seat {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seats := make([]*model.Seat, 0, len(r.data))
	for _, seat := range r.data {
		// Skip expired locks
		if seat.IsExpired() {
			continue
		}
		seats = append(seats, seat)
	}
	return seats
}
