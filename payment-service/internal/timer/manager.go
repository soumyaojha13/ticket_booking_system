package timer

import (
	"context"
	"sync"
	"time"
)

// Manager owns seat-hold timers (5 minutes) and allows cancellation on success.
type Manager struct {
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

func NewManager() *Manager {
	return &Manager{cancels: make(map[string]context.CancelFunc)}
}

// Start starts (or replaces) a timer for a seat. When the timer fires, onExpire is called.
func (m *Manager) Start(seatID string, d time.Duration, onExpire func()) {
	if seatID == "" {
		return
	}

	m.mu.Lock()
	// Replace any existing timer for the seat.
	if cancel, ok := m.cancels[seatID]; ok {
		cancel()
		delete(m.cancels, seatID)
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancels[seatID] = cancel
	m.mu.Unlock()

	go func() {
		select {
		case <-time.After(d):
			if onExpire != nil {
				onExpire()
			}
			m.Stop(seatID)
		case <-ctx.Done():
			return
		}
	}()
}

// Stop cancels and removes the timer for a seat (if any).
func (m *Manager) Stop(seatID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cancel, ok := m.cancels[seatID]; ok {
		cancel()
		delete(m.cancels, seatID)
	}
}

