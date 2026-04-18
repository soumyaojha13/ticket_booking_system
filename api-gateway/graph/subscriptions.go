package graph

import (
	"context"
	"io"
	"log"
	"sync"
	"time"

	"github.com/soumyaojha/ticket-booking-system/api-gateway/graph/model"
)

func (r *Resolver) subscribeAll(ctx context.Context) (<-chan *model.SeatEvent, func()) {
	ch := make(chan *model.SeatEvent, 32)
	id := r.nextSubID.Add(1)
	r.allSubs.Store(id, ch)

	cleanup := func() {
		r.allSubs.Delete(id)
		close(ch)
	}

	go func() {
		<-ctx.Done()
		cleanup()
	}()

	return ch, cleanup
}

func (r *Resolver) subscribeSeat(ctx context.Context, seatID string) (<-chan *model.SeatEvent, func()) {
	ch := make(chan *model.SeatEvent, 32)
	id := r.nextSubID.Add(1)

	mAny, _ := r.seatSubs.LoadOrStore(seatID, &syncMap{})
	m := mAny.(*syncMap)
	m.Store(id, ch)

	cleanup := func() {
		m.Delete(id)
		close(ch)
	}

	go func() {
		<-ctx.Done()
		cleanup()
	}()

	return ch, cleanup
}

// publish pushes an event to GraphQL subscribers (non-blocking).
func (r *Resolver) publish(event *model.SeatEvent) {
	r.allSubs.Range(func(_, v any) bool {
		if ch, ok := v.(chan *model.SeatEvent); ok {
			select {
			case ch <- event:
			default:
			}
		}
		return true
	})

	if event == nil {
		return
	}
	if mAny, ok := r.seatSubs.Load(event.SeatID); ok {
		m := mAny.(*syncMap)
		m.Range(func(_, v any) bool {
			if ch, ok := v.(chan *model.SeatEvent); ok {
				select {
				case ch <- event:
				default:
				}
			}
			return true
		})
	}
}

// StartSeatStream continuously reads Inventory SeatStream and republishes events to GraphQL subscriptions.
// This is the bridge from gRPC streaming to GraphQL subscriptions.
func (r *Resolver) StartSeatStream(ctx context.Context) {
	if r.Inventory == nil {
		log.Println("SeatStream bridge not started: inventory client is nil")
		return
	}

	backoff := 500 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		stream, err := r.Inventory.SeatStream(ctx)
		if err != nil {
			log.Printf("SeatStream connect error: %v\n", err)
			time.Sleep(backoff)
			if backoff < 5*time.Second {
				backoff *= 2
			}
			continue
		}

		// Reset backoff after successful connect.
		backoff = 500 * time.Millisecond

		for {
			ev, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("SeatStream recv error: %v\n", err)
				break
			}
			if ev == nil || ev.SeatId == "" || ev.Status == "" {
				continue
			}

			gql := &model.SeatEvent{
				SeatID:      ev.SeatId,
				Status:      toGqlSeatStatus(ev.Status),
				TimestampMs: int(ev.TimestampMs),
			}
			if ev.UserId != "" {
				u := ev.UserId
				gql.UserID = &u
			}
			r.publish(gql)
		}
	}
}

func toGqlSeatStatus(s string) model.SeatStatus {
	switch s {
	case "LOCKED":
		return model.SeatStatusLocked
	case "BOOKED":
		return model.SeatStatusBooked
	default:
		return model.SeatStatusAvailable
	}
}

// Minimal sync-map wrapper so we can type-assert cleanly.
type syncMap struct{ m sync.Map }

func (s *syncMap) Store(k, v any)        { s.m.Store(k, v) }
func (s *syncMap) Delete(k any)          { s.m.Delete(k) }
func (s *syncMap) Range(f func(any, any) bool) { s.m.Range(f) }

