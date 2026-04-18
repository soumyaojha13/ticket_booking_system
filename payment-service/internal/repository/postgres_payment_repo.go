package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/model"
)

type postgresPaymentRepo struct {
	db *sql.DB
}

func NewPostgresPaymentRepo(db *sql.DB) (PaymentRepository, error) {
	repo := &postgresPaymentRepo{db: db}
	if err := repo.ensureSchema(context.Background()); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *postgresPaymentRepo) Create(payment *model.Payment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO payments (seat_id, user_id, amount, status, created_at, expires_at, transaction_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		payment.SeatID,
		payment.UserID,
		payment.Amount,
		payment.Status,
		payment.CreatedAt,
		payment.ExpiresAt,
		nullIfEmpty(payment.TransactionID),
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return errors.New("payment already exists for this seat")
		}
		return err
	}
	return nil
}

func (r *postgresPaymentRepo) Get(seatID string) (*model.Payment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(
		ctx,
		`SELECT seat_id, user_id, amount, status, created_at, expires_at, COALESCE(transaction_id, '')
		 FROM payments WHERE seat_id = $1`,
		seatID,
	)

	var payment model.Payment
	if err := row.Scan(
		&payment.SeatID,
		&payment.UserID,
		&payment.Amount,
		&payment.Status,
		&payment.CreatedAt,
		&payment.ExpiresAt,
		&payment.TransactionID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPaymentNotFound
		}
		return nil, err
	}
	return &payment, nil
}

func (r *postgresPaymentRepo) Update(payment *model.Payment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.db.ExecContext(
		ctx,
		`UPDATE payments
		 SET user_id = $2, amount = $3, status = $4, created_at = $5, expires_at = $6, transaction_id = $7
		 WHERE seat_id = $1`,
		payment.SeatID,
		payment.UserID,
		payment.Amount,
		payment.Status,
		payment.CreatedAt,
		payment.ExpiresAt,
		nullIfEmpty(payment.TransactionID),
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

func (r *postgresPaymentRepo) Delete(seatID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.db.ExecContext(ctx, `DELETE FROM payments WHERE seat_id = $1`, seatID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

func (r *postgresPaymentRepo) GetAll() []*model.Payment {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT seat_id, user_id, amount, status, created_at, expires_at, COALESCE(transaction_id, '')
		 FROM payments`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	payments := make([]*model.Payment, 0)
	for rows.Next() {
		var payment model.Payment
		if err := rows.Scan(
			&payment.SeatID,
			&payment.UserID,
			&payment.Amount,
			&payment.Status,
			&payment.CreatedAt,
			&payment.ExpiresAt,
			&payment.TransactionID,
		); err != nil {
			continue
		}
		if !payment.IsExpired() {
			payments = append(payments, &payment)
		}
	}

	return payments
}

func (r *postgresPaymentRepo) ensureSchema(ctx context.Context) error {
	schemaCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(
		schemaCtx,
		`CREATE TABLE IF NOT EXISTS payments (
			seat_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			amount INTEGER NOT NULL,
			status TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			transaction_id TEXT
		)`,
	)
	return err
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
