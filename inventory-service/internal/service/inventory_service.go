// internal/service/inventory_service.go
package service

import (
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"context"
)

type InventoryService struct {
	redis *redis.Client
}

func NewInventoryService(r *redis.Client) *InventoryService {
	return &InventoryService{redis: r}
}

// Lock seat for 5 minutes
func (s *InventoryService) LockSeat(seatID string) error {
	key := fmt.Sprintf("seat:%s", seatID)

	// SETNX ensures no double booking
	ok, err := s.redis.SetNX(context.Background(), key, "locked", 5*time.Minute).Result()
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("seat already locked")
	}

	return nil
}

// Unlock seat manually (on payment failure)
func (s *InventoryService) UnlockSeat(seatID string) error {
	key := fmt.Sprintf("seat:%s", seatID)
	return s.redis.Del(context.Background(), key).Err()
}

// Confirm booking (remove lock permanently)
func (s *InventoryService) ConfirmSeat(seatID string) error {
	key := fmt.Sprintf("seat:%s", seatID)
	return s.redis.Del(context.Background(), key).Err()
}