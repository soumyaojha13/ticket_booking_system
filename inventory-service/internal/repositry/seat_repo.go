//go:build ignore

package repository

import (
	"sync"

	"inventory-service/internal/model"
)

type SeatRepository interface {
	Get(seatID string) (*model.Seat, bool)
	Save(seat *model.Seat)
}

type inMemoryRepo struct {
	data map[string]*model.Seat
	mu   sync.RWMutex
}

func NewSeatRepo() SeatRepository {
	return &inMemoryRepo{
		data: make(map[string]*model.Seat),
	}
}

func (r *inMemoryRepo) Get(seatID string) (*model.Seat, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seat, ok := r.data[seatID]
	return seat, ok
}

func (r *inMemoryRepo) Save(seat *model.Seat) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[seat.ID] = seat
}
