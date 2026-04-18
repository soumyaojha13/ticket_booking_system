package handler

import (
	"context"
	"io"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/soumyaojha/ticket-booking-system/inventory-service/internal/usecase"
	seatv1 "github.com/soumyaojha/ticket-booking-system/proto/seat/v1"
)

type SeatHandler struct {
	seatv1.UnimplementedSeatServiceServer
	usecase       usecase.SeatUsecase
	streamClients sync.Map // Map[string][]chan *seatv1.SeatEvent for broadcasting
}

func NewSeatHandler(u usecase.SeatUsecase) *SeatHandler {
	return &SeatHandler{usecase: u}
}

// LockSeat implements the LockSeat RPC with context timeout
func (h *SeatHandler) LockSeat(ctx context.Context, req *seatv1.LockSeatRequest) (*seatv1.SeatResponse, error) {
	// Use context timeout from client
	seat, err := h.usecase.LockSeat(ctx, req.SeatId, req.UserId)
	if err != nil {
		return &seatv1.SeatResponse{
			SeatId: req.SeatId,
			Status: "ERROR",
		}, err
	}

	// Broadcast seat event to all subscribers
	h.broadcastSeatEvent(&seatv1.SeatEvent{
		SeatId:      seat.ID,
		Status:      seat.Status,
		UserId:      seat.UserID,
		TimestampMs: time.Now().UnixMilli(),
	})

	return &seatv1.SeatResponse{
		SeatId:      seat.ID,
		Status:      seat.Status,
		UserId:      seat.UserID,
		LockedAtMs:  seat.LockedAt.UnixMilli(),
		ExpiresAtMs: seat.ExpiresAt.UnixMilli(),
	}, nil
}

// ReleaseSeat implements the ReleaseSeat RPC
func (h *SeatHandler) ReleaseSeat(ctx context.Context, req *seatv1.ReleaseSeatRequest) (*seatv1.SeatResponse, error) {
	seat, err := h.usecase.ReleaseSeat(ctx, req.SeatId, req.Reason)
	if err != nil {
		return &seatv1.SeatResponse{
			SeatId: req.SeatId,
			Status: "ERROR",
		}, err
	}

	// Broadcast seat event
	h.broadcastSeatEvent(&seatv1.SeatEvent{
		SeatId:      seat.ID,
		Status:      seat.Status,
		UserId:      seat.UserID,
		TimestampMs: time.Now().UnixMilli(),
	})

	return &seatv1.SeatResponse{
		SeatId: seat.ID,
		Status: seat.Status,
	}, nil
}

// BookSeat implements the BookSeat RPC to confirm booking
func (h *SeatHandler) BookSeat(ctx context.Context, req *seatv1.BookSeatRequest) (*seatv1.SeatResponse, error) {
	seat, err := h.usecase.BookSeat(ctx, req.SeatId, req.UserId)
	if err != nil {
		return &seatv1.SeatResponse{
			SeatId: req.SeatId,
			Status: "ERROR",
		}, err
	}

	// Broadcast seat event
	h.broadcastSeatEvent(&seatv1.SeatEvent{
		SeatId:      seat.ID,
		Status:      seat.Status,
		UserId:      seat.UserID,
		TimestampMs: time.Now().UnixMilli(),
	})

	return &seatv1.SeatResponse{
		SeatId: seat.ID,
		Status: seat.Status,
		UserId: seat.UserID,
	}, nil
}

// GetSeatStatus implements the GetSeatStatus RPC
func (h *SeatHandler) GetSeatStatus(ctx context.Context, req *seatv1.SeatStatusRequest) (*seatv1.SeatResponse, error) {
	seat, err := h.usecase.GetSeatStatus(ctx, req.SeatId)
	if err != nil {
		return &seatv1.SeatResponse{
			SeatId: req.SeatId,
			Status: "ERROR",
		}, err
	}

	return &seatv1.SeatResponse{
		SeatId:      seat.ID,
		Status:      seat.Status,
		UserId:      seat.UserID,
		LockedAtMs:  seat.LockedAt.UnixMilli(),
		ExpiresAtMs: seat.ExpiresAt.UnixMilli(),
	}, nil
}

// SeatStream implements bi-directional streaming for real-time seat updates
func (h *SeatHandler) SeatStream(stream seatv1.SeatService_SeatStreamServer) error {
	ctx := stream.Context()

	// Create a channel for this client
	eventChan := make(chan *seatv1.SeatEvent, 10) // Buffered channel
	clientID := generateClientID()

	// Register this client
	h.streamClients.Store(clientID, eventChan)
	defer func() {
		h.streamClients.Delete(clientID)
		close(eventChan)
	}()

	// Run two goroutines - one for receiving, one for sending
	errChan := make(chan error, 1)

	// Goroutine 1: Receive from client
	go func() {
		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				_, err := stream.Recv()
				if err == io.EOF {
					errChan <- nil
					return
				}
				if err != nil {
					errChan <- err
					return
				}
			}
		}
	}()

	// Goroutine 2: Send to client
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-eventChan:
			if err := stream.Send(event); err != nil {
				return err
			}
		case err := <-errChan:
			return err
		}
	}
}

// broadcastSeatEvent sends a seat event to all connected clients
func (h *SeatHandler) broadcastSeatEvent(event *seatv1.SeatEvent) {
	h.streamClients.Range(func(key, value interface{}) bool {
		eventChan, ok := value.(chan *seatv1.SeatEvent)
		if !ok {
			return true
		}

		// Non-blocking send to avoid deadlocks
		select {
		case eventChan <- event:
		default:
			log.Printf("Event channel full for client %v, dropping event\n", key)
		}
		return true
	})
}

var counter int
var counterMu sync.Mutex

func generateClientID() string {
	counterMu.Lock()
	defer counterMu.Unlock()
	counter++
	return "client_" + strconv.Itoa(counter)
}
