//go:build ignore

package repository

import (
	"context"
	"errors"
	"time"

	"github.com/soumyaojha/ticket-booking-system/inventory-service/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoSeatRepo struct {
	collection *mongo.Collection
	timeout    time.Duration
}

type seatDocument struct {
	ID        string    `bson:"_id"`
	Status    string    `bson:"status"`
	UserID    string    `bson:"user_id"`
	LockedAt  time.Time `bson:"locked_at,omitempty"`
	ExpiresAt time.Time `bson:"expires_at,omitempty"`
}

func NewMongoSeatRepo(collection *mongo.Collection) SeatRepository {
	return &mongoSeatRepo{
		collection: collection,
		timeout:    5 * time.Second,
	}
}

func (r *mongoSeatRepo) context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), r.timeout)
}

func (r *mongoSeatRepo) Get(seatID string) (*model.Seat, bool) {
	ctx, cancel := r.context()
	defer cancel()

	var doc seatDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": seatID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, false
	}
	if err != nil {
		return nil, false
	}

	seat := doc.toModel()
	if seat.IsExpired() {
		return &model.Seat{
			ID:     seat.ID,
			Status: "AVAILABLE",
			UserID: "",
		}, true
	}

	return seat, true
}

func (r *mongoSeatRepo) Save(seat *model.Seat) {
	ctx, cancel := r.context()
	defer cancel()

	_, _ = r.collection.ReplaceOne(
		ctx,
		bson.M{"_id": seat.ID},
		seatDocumentFromModel(seat),
		options.Replace().SetUpsert(true),
	)
}

func (r *mongoSeatRepo) Lock(seatID, userID string, lockDuration time.Duration) (*model.Seat, error) {
	ctx, cancel := r.context()
	defer cancel()

	now := time.Now()
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{
			"_id": seatID,
			"$or": bson.A{
				bson.M{"status": bson.M{"$ne": "LOCKED"}},
				bson.M{"expires_at": bson.M{"$lte": now}},
			},
		},
		bson.M{
			"$set": bson.M{
				"status":     "LOCKED",
				"user_id":    userID,
				"locked_at":  now,
				"expires_at": now.Add(lockDuration),
			},
		},
		options.UpdateOne().SetUpsert(true),
	)
	if mongo.IsDuplicateKeyError(err) {
		return nil, ErrSeatAlreadyLocked
	}
	if err != nil {
		return nil, err
	}

	seat, ok := r.Get(seatID)
	if !ok {
		return nil, ErrSeatNotFound
	}
	return seat, nil
}

func (r *mongoSeatRepo) Release(seatID string, reason string) (*model.Seat, error) {
	ctx, cancel := r.context()
	defer cancel()

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": seatID},
		bson.M{
			"$set": bson.M{
				"status":     "AVAILABLE",
				"user_id":    "",
				"locked_at":  time.Time{},
				"expires_at": time.Time{},
			},
		},
	)
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, ErrSeatNotFound
	}

	seat, ok := r.Get(seatID)
	if !ok {
		return nil, ErrSeatNotFound
	}
	return seat, nil
}

func (r *mongoSeatRepo) Book(seatID, userID string) (*model.Seat, error) {
	seat, ok := r.Get(seatID)
	if !ok {
		return nil, ErrSeatNotFound
	}

	if seat.Status == "LOCKED" && seat.UserID != userID {
		return nil, ErrSeatLockedByOtherUser
	}

	ctx, cancel := r.context()
	defer cancel()

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": seatID},
		bson.M{
			"$set": bson.M{
				"status":     "BOOKED",
				"user_id":    userID,
				"locked_at":  seat.LockedAt,
				"expires_at": seat.ExpiresAt,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	if result.MatchedCount == 0 {
		return nil, ErrSeatNotFound
	}

	updatedSeat, exists := r.Get(seatID)
	if !exists {
		return nil, ErrSeatNotFound
	}

	return updatedSeat, nil
}

func (r *mongoSeatRepo) GetAll() []*model.Seat {
	ctx, cancel := r.context()
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil
	}
	defer cursor.Close(ctx)

	seats := make([]*model.Seat, 0)
	for cursor.Next(ctx) {
		var doc seatDocument
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		seat := doc.toModel()
		if seat.IsExpired() {
			continue
		}
		seats = append(seats, seat)
	}

	return seats
}

func seatDocumentFromModel(seat *model.Seat) seatDocument {
	return seatDocument{
		ID:        seat.ID,
		Status:    seat.Status,
		UserID:    seat.UserID,
		LockedAt:  seat.LockedAt,
		ExpiresAt: seat.ExpiresAt,
	}
}

func (d seatDocument) toModel() *model.Seat {
	return &model.Seat{
		ID:        d.ID,
		Status:    d.Status,
		UserID:    d.UserID,
		LockedAt:  d.LockedAt,
		ExpiresAt: d.ExpiresAt,
	}
}
