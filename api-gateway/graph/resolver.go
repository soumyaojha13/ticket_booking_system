package graph

import (
	"sync"
	"sync/atomic"

	"github.com/soumyaojha/ticket-booking-system/api-gateway/internal/grpc"
)

type Resolver struct {
	Inventory *grpc.InventoryClient
	Payment   *grpc.PaymentClient

	// Subscription fanout (GraphQL subscriptions).
	nextSubID atomic.Int64
	allSubs   sync.Map // id(int64) -> chan *SeatEvent
	seatSubs  sync.Map // key(string seatId) -> per-seat sync-map
}

