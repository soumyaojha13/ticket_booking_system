package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
	"github.com/soumyaojha/ticket-booking-system/inventory-service/internal/model"
)

type hybridSeatRepo struct {
	redisRepo SeatRepository
	db        *sql.DB
}

func NewHybridSeatRepo(redisRepo SeatRepository, db *sql.DB) (SeatRepository, error) {
	repo := &hybridSeatRepo{
		redisRepo: redisRepo,
		db:        db,
	}

	if err := repo.ensureSchema(context.Background()); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *hybridSeatRepo) Get(seatID string) (*model.Seat, bool) {
	if seat, ok := r.redisRepo.Get(seatID); ok {
		return seat, true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(
		ctx,
		`SELECT seat_id, user_id, booked_at
		 FROM booked_seats
		 WHERE seat_id = $1`,
		seatID,
	)

	var seat model.Seat
	var bookedAt time.Time
	if err := row.Scan(&seat.ID, &seat.UserID, &bookedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false
		}
		return nil, false
	}

	seat.Status = "BOOKED"
	seat.LockedAt = bookedAt
	return &seat, true
}

func (r *hybridSeatRepo) Save(seat *model.Seat) {
	r.redisRepo.Save(seat)
	if seat.Status == "BOOKED" {
		_ = r.persistBooking(seat)
	}
}

func (r *hybridSeatRepo) Lock(seatID, userID string, lockDuration time.Duration) (*model.Seat, error) {
	if seat, ok := r.Get(seatID); ok && seat.Status == "BOOKED" {
		return nil, ErrSeatNotAvailable
	}

	return r.redisRepo.Lock(seatID, userID, lockDuration)
}

func (r *hybridSeatRepo) Release(seatID string, reason string) (*model.Seat, error) {
	return r.redisRepo.Release(seatID, reason)
}

func (r *hybridSeatRepo) Book(seatID, userID string) (*model.Seat, error) {
	if seat, ok := r.Get(seatID); ok && seat.Status == "BOOKED" {
		return nil, ErrSeatNotAvailable
	}

	seat, err := r.redisRepo.Book(seatID, userID)
	if err != nil {
		return nil, err
	}

	if err := r.persistBooking(seat); err != nil {
		return nil, err
	}

	return seat, nil
}

func (r *hybridSeatRepo) GetAll() []*model.Seat {
	seen := make(map[string]struct{})
	seats := make([]*model.Seat, 0)

	for _, seat := range r.redisRepo.GetAll() {
		seats = append(seats, seat)
		seen[seat.ID] = struct{}{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT seat_id, user_id, booked_at
		 FROM booked_seats`,
	)
	if err != nil {
		return seats
	}
	defer rows.Close()

	for rows.Next() {
		var seat model.Seat
		var bookedAt time.Time
		if err := rows.Scan(&seat.ID, &seat.UserID, &bookedAt); err != nil {
			continue
		}
		if _, exists := seen[seat.ID]; exists {
			continue
		}
		seat.Status = "BOOKED"
		seat.LockedAt = bookedAt
		seats = append(seats, &seat)
	}

	return seats
}

func (r *hybridSeatRepo) ensureSchema(ctx context.Context) error {
	schemaCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(
		schemaCtx,
		`CREATE TABLE IF NOT EXISTS booked_seats (
			seat_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			booked_at TIMESTAMPTZ NOT NULL
		)`,
	)
	return err
}

func (r *hybridSeatRepo) persistBooking(seat *model.Seat) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bookedAt := seat.LockedAt
	if bookedAt.IsZero() {
		bookedAt = time.Now()
	}

	result, err := r.db.ExecContext(
		ctx,
		`INSERT INTO booked_seats (seat_id, user_id, booked_at)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (seat_id) DO NOTHING`,
		seat.ID,
		seat.UserID,
		bookedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrSeatNotAvailable
		}
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSeatNotAvailable
	}

	return nil
}
