package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/soumyaojha/ticket-booking-system/inventory-service/internal/model"
)

type redisSeatRepo struct {
	client *redis.Client
}

func NewRedisSeatRepo(client *redis.Client) SeatRepository {
	return &redisSeatRepo{client: client}
}

func (r *redisSeatRepo) Get(seatID string) (*model.Seat, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	value, err := r.client.Get(ctx, r.key(seatID)).Result()
	if err == redis.Nil {
		return nil, false
	}
	if err != nil {
		return nil, false
	}

	seat, err := decodeSeat(value)
	if err != nil {
		return nil, false
	}

	if seat.IsExpired() {
		_, _ = r.Release(seatID, "LOCK_EXPIRED")
		return &model.Seat{ID: seatID, Status: "AVAILABLE"}, true
	}

	return seat, true
}

func (r *redisSeatRepo) Save(seat *model.Seat) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	payload, err := encodeSeat(seat)
	if err != nil {
		return
	}

	ttl := seatTTL(seat)
	_ = r.client.Set(ctx, r.key(seat.ID), payload, ttl).Err()
}

func (r *redisSeatRepo) Lock(seatID, userID string, lockDuration time.Duration) (*model.Seat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	seat := &model.Seat{
		ID:        seatID,
		Status:    "LOCKED",
		UserID:    userID,
		LockedAt:  now,
		ExpiresAt: now.Add(lockDuration),
	}

	payload, err := encodeSeat(seat)
	if err != nil {
		return nil, err
	}

	ok, err := r.client.SetNX(ctx, r.key(seatID), payload, lockDuration).Result()
	if err != nil {
		return nil, err
	}
	if ok {
		return seat, nil
	}

	existing, exists := r.Get(seatID)
	if exists && existing.Status == "LOCKED" && !existing.IsExpired() {
		return nil, ErrSeatAlreadyLocked
	}

	_ = r.client.Del(ctx, r.key(seatID)).Err()
	ok, err = r.client.SetNX(ctx, r.key(seatID), payload, lockDuration).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrSeatAlreadyLocked
	}

	return seat, nil
}

func (r *redisSeatRepo) Release(seatID string, reason string) (*model.Seat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	seat, exists := r.Get(seatID)
	if !exists {
		return nil, ErrSeatNotFound
	}

	seat.Status = "AVAILABLE"
	seat.UserID = ""
	seat.LockedAt = time.Time{}
	seat.ExpiresAt = time.Time{}

	payload, err := encodeSeat(seat)
	if err != nil {
		return nil, err
	}

	if err := r.client.Set(ctx, r.key(seatID), payload, 0).Err(); err != nil {
		return nil, err
	}
	return seat, nil
}

func (r *redisSeatRepo) Book(seatID, userID string) (*model.Seat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	seat, exists := r.Get(seatID)
	if !exists {
		return nil, ErrSeatNotFound
	}

	if seat.Status == "LOCKED" && seat.UserID != userID {
		return nil, ErrSeatLockedByOtherUser
	}

	seat.Status = "BOOKED"
	seat.UserID = userID
	payload, err := encodeSeat(seat)
	if err != nil {
		return nil, err
	}

	if err := r.client.Set(ctx, r.key(seatID), payload, 0).Err(); err != nil {
		return nil, err
	}
	return seat, nil
}

func (r *redisSeatRepo) GetAll() []*model.Seat {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keys, err := r.client.Keys(ctx, "seat:*").Result()
	if err != nil {
		return nil
	}

	seats := make([]*model.Seat, 0, len(keys))
	for _, key := range keys {
		value, err := r.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		seat, err := decodeSeat(value)
		if err != nil || seat.IsExpired() {
			continue
		}
		seats = append(seats, seat)
	}
	return seats
}

func (r *redisSeatRepo) key(seatID string) string {
	return "seat:" + seatID
}

func encodeSeat(seat *model.Seat) (string, error) {
	payload, err := json.Marshal(seat)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func decodeSeat(value string) (*model.Seat, error) {
	var seat model.Seat
	if err := json.Unmarshal([]byte(value), &seat); err != nil {
		return nil, err
	}
	return &seat, nil
}

func seatTTL(seat *model.Seat) time.Duration {
	if seat.Status == "LOCKED" && !seat.ExpiresAt.IsZero() {
		ttl := time.Until(seat.ExpiresAt)
		if ttl > 0 {
			return ttl
		}
	}
	return 0
}
