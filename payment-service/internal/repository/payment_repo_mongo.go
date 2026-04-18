//go:build ignore

package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const paymentCollectionName = "payments"

type mongoPaymentRepo struct {
	collection *mongo.Collection
}

func NewMongoPaymentRepo(collection *mongo.Collection) (PaymentRepository, error) {
	repo := &mongoPaymentRepo{
		collection: collection,
	}

	if err := repo.ensureIndexes(context.Background()); err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *mongoPaymentRepo) ensureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "seatId", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create payment indexes: %w", err)
	}
	return nil
}

func (r *mongoPaymentRepo) Create(payment *model.Payment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.InsertOne(ctx, payment)
	if mongo.IsDuplicateKeyError(err) {
		return errors.New("payment already exists for this seat")
	}
	return err
}

func (r *mongoPaymentRepo) Get(seatID string) (*model.Payment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var payment model.Payment
	err := r.collection.FindOne(ctx, bson.M{"seatId": seatID}).Decode(&payment)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrPaymentNotFound
	}
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

func (r *mongoPaymentRepo) Update(payment *model.Payment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.collection.ReplaceOne(ctx, bson.M{"seatId": payment.SeatID}, payment)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

func (r *mongoPaymentRepo) Delete(seatID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.collection.DeleteOne(ctx, bson.M{"seatId": seatID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

func (r *mongoPaymentRepo) GetAll() []*model.Payment {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{
		"expiresAt": bson.M{"$gt": time.Now()},
	})
	if err != nil {
		return nil
	}
	defer cursor.Close(ctx)

	var payments []*model.Payment
	for cursor.Next(ctx) {
		var payment model.Payment
		if err := cursor.Decode(&payment); err == nil {
			payments = append(payments, &payment)
		}
	}
	return payments
}
