//go:build ignore

package repository

import (
	"context"
	"errors"
	"time"

	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoPaymentRepo struct {
	collection *mongo.Collection
	timeout    time.Duration
}

type paymentDocument struct {
	SeatID        string    `bson:"_id"`
	UserID        string    `bson:"user_id"`
	Amount        int32     `bson:"amount"`
	Status        string    `bson:"status"`
	CreatedAt     time.Time `bson:"created_at"`
	ExpiresAt     time.Time `bson:"expires_at"`
	TransactionID string    `bson:"transaction_id,omitempty"`
}

func NewMongoPaymentRepo(collection *mongo.Collection) PaymentRepository {
	return &mongoPaymentRepo{
		collection: collection,
		timeout:    5 * time.Second,
	}
}

func (r *mongoPaymentRepo) context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), r.timeout)
}

func (r *mongoPaymentRepo) Create(payment *model.Payment) error {
	ctx, cancel := r.context()
	defer cancel()

	_, err := r.collection.InsertOne(ctx, paymentDocumentFromModel(payment))
	if mongo.IsDuplicateKeyError(err) {
		return errors.New("payment already exists for this seat")
	}
	return err
}

func (r *mongoPaymentRepo) Get(seatID string) (*model.Payment, error) {
	ctx, cancel := r.context()
	defer cancel()

	var doc paymentDocument
	err := r.collection.FindOne(ctx, bson.M{"_id": seatID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrPaymentNotFound
	}
	if err != nil {
		return nil, err
	}

	return doc.toModel(), nil
}

func (r *mongoPaymentRepo) Update(payment *model.Payment) error {
	ctx, cancel := r.context()
	defer cancel()

	result, err := r.collection.ReplaceOne(
		ctx,
		bson.M{"_id": payment.SeatID},
		paymentDocumentFromModel(payment),
		options.Replace(),
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

func (r *mongoPaymentRepo) Delete(seatID string) error {
	ctx, cancel := r.context()
	defer cancel()

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": seatID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrPaymentNotFound
	}
	return nil
}

func (r *mongoPaymentRepo) GetAll() []*model.Payment {
	ctx, cancel := r.context()
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil
	}
	defer cursor.Close(ctx)

	payments := make([]*model.Payment, 0)
	for cursor.Next(ctx) {
		var doc paymentDocument
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		payment := doc.toModel()
		if !payment.IsExpired() {
			payments = append(payments, payment)
		}
	}

	return payments
}

func paymentDocumentFromModel(payment *model.Payment) paymentDocument {
	return paymentDocument{
		SeatID:        payment.SeatID,
		UserID:        payment.UserID,
		Amount:        payment.Amount,
		Status:        payment.Status,
		CreatedAt:     payment.CreatedAt,
		ExpiresAt:     payment.ExpiresAt,
		TransactionID: payment.TransactionID,
	}
}

func (d paymentDocument) toModel() *model.Payment {
	return &model.Payment{
		SeatID:        d.SeatID,
		UserID:        d.UserID,
		Amount:        d.Amount,
		Status:        d.Status,
		CreatedAt:     d.CreatedAt,
		ExpiresAt:     d.ExpiresAt,
		TransactionID: d.TransactionID,
	}
}
