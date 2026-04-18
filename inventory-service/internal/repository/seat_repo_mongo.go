//go:build ignore

package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/soumyaojha/ticket-booking-system/inventory-service/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const seatCollectionName = "seats"

type mongoSeatRepo struct {
	collection *mongo.Collection
}

func NewMongoSeatRepo(collection *mongo.Collection) (SeatRepository, error) {
	repo := &mongoSeatRepo{
		collection: collection,
	}

	if err := repo.ensureIndexes(context.Background()); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *mongoSeatRepo) ensureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}, {Key: "expiresAt", Value: 1}},
		},
	})
	if err != nil {
		return fmt.Errorf("create seat indexes: %w", err)
	}
	return nil
}

func (r *mongoSeatRepo) Get(seatID string) (*model.Seat, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var seat model.Seat
	err := r.collection.FindOne(ctx, bson.M{"id": seatID}).Decode(&seat)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, false
	}
	if err != nil {
		return nil, false
	}

	if seat.IsExpired() {
		if _, releaseErr := r.Release(seatID, "LOCK_EXPIRED"); releaseErr != nil {
			return &model.Seat{ID: seatID, Status: "AVAILABLE"}, true
		}
		return &model.Seat{ID: seatID, Status: "AVAILABLE"}, true
	}

	return &seat, true
}

func (r *mongoSeatRepo) Save(seat *model.Seat) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _ = r.collection.ReplaceOne(
		ctx,
		bson.M{"id": seat.ID},
		seat,
		options.Replace().SetUpsert(true),
	)
}

func (r *mongoSeatRepo) Lock(seatID, userID string, lockDuration time.Duration) (*model.Seat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()

	filter := bson.M{
		"id": seatID,
		"$or": []bson.M{
			{"status": bson.M{"$ne": "LOCKED"}},
			{"expiresAt": bson.M{"$lte": now}},
		},
	}
	update := bson.M{
		"$set": bson.M{
			"id":        seatID,
			"status":    "LOCKED",
			"userId":    userID,
			"lockedAt":  now,
			"expiresAt": now.Add(lockDuration),
		},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var seat model.Seat
	err := r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&seat)
	if err == nil {
		return &seat, nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, err
	}

	existing, ok := r.Get(seatID)
	if ok && existing.Status == "LOCKED" && !existing.IsExpired() {
		return nil, ErrSeatAlreadyLocked
	}

	return nil, ErrSeatAlreadyLocked
}

func (r *mongoSeatRepo) Release(seatID string, reason string) (*model.Seat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"status":    "AVAILABLE",
			"userId":    "",
			"lockedAt":  time.Time{},
			"expiresAt": time.Time{},
		},
	}
	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After)

	var seat model.Seat
	if err := r.collection.FindOneAndUpdate(ctx, bson.M{"id": seatID}, update, opts).Decode(&seat); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrSeatNotFound
		}
		return nil, err
	}

	return &seat, nil
}

func (r *mongoSeatRepo) Book(seatID, userID string) (*model.Seat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	filter := bson.M{
		"id": seatID,
		"$or": []bson.M{
			{"status": bson.M{"$ne": "LOCKED"}},
			{"userId": userID},
			{"expiresAt": bson.M{"$lte": now}},
		},
	}
	update := bson.M{
		"$set": bson.M{
			"id":        seatID,
			"status":    "BOOKED",
			"userId":    userID,
			"lockedAt":  time.Time{},
			"expiresAt": time.Time{},
		},
	}

	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After)

	var seat model.Seat
	err := r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&seat)
	if err == nil {
		return &seat, nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, err
	}

	existing, ok := r.Get(seatID)
	if !ok {
		return nil, ErrSeatNotFound
	}
	if existing.Status == "LOCKED" && existing.UserID != userID && !existing.IsExpired() {
		return nil, ErrSeatLockedByOtherUser
	}

	return nil, ErrSeatNotFound
}

func (r *mongoSeatRepo) GetAll() []*model.Seat {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{
		"$or": []bson.M{
			{"status": bson.M{"$ne": "LOCKED"}},
			{"expiresAt": bson.M{"$gt": time.Now()}},
		},
	})
	if err != nil {
		return nil
	}
	defer cursor.Close(ctx)

	var seats []*model.Seat
	for cursor.Next(ctx) {
		var seat model.Seat
		if err := cursor.Decode(&seat); err == nil {
			seats = append(seats, &seat)
		}
	}
	return seats
}
