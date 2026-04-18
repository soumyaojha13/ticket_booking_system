# Ticket Booking System - Implementation Guide

## Overview

This is a complete ticket booking system for a world tour with the following key features:
- 5-minute seat lock mechanism
- Automatic seat expiry using goroutines and time.After
- Bidirectional gRPC streaming for real-time seat updates
- GraphQL subscriptions for frontend updates
- Worker pool with buffered channels for handling flash sale spikes
- Context timeouts on all gRPC calls to prevent hanging

## Architecture

### Three Main Services

#### 1. **Inventory Service** (Port 50051)
- Manages available seats
- Handles seat locking (5 minutes)
- Broadcasts seat status updates via bidirectional streaming
- **Key Components:**
  - `SeatRepository`: Thread-safe in-memory seat storage with lock expiry tracking
  - `SeatUsecase`: Business logic for locking, releasing, and booking seats
  - `SeatHandler`: gRPC service implementation with streaming support

#### 2. **Payment Service** (Port 50052)
- Manages 5-minute payment timers
- Processes payments with worker pool
- Confirms bookings after successful payment
- **Key Components:**
  - `PaymentRepository`: Manages pending payments with expiry tracking
  - `PaymentUsecase`: Business logic for payment timers
  - `PaymentHandler`: gRPC service implementation
  - `WorkerPool`: Buffered channel-based worker pool for concurrent payment processing

#### 3. **API Gateway** (Port 8080)
- GraphQL endpoint for clients
- Manages gRPC calls to both services
- GraphQL subscriptions for real-time updates
- **Key Features:**
  - Context timeouts on all gRPC calls
  - Bi-directional service communication

## Key Implementation Details

### 1. Channels & Timers (Goroutines)

**Location:** `inventory-service/internal/usecase/seat_usecase.go` and `payment-service/internal/usecase/payment_usecase.go`

```go
// Uses time.After inside goroutines
select {
case <-time.After(SEAT_LOCK_DURATION):
    // Lock expired - auto-release seat
    u.ReleaseSeat(ctx, seatID, "TIMEOUT")
case <-timerCtx.Done():
    // Lock was manually released or booking confirmed
}
```

**Features:**
- ✅ `time.After()` used for 5-minute timeout
- ✅ Goroutines manage individual seat timers
- ✅ Context cancellation for graceful shutdown
- ✅ Concurrent timer management (one per seat)

### 2. Context Timeouts on gRPC Calls

**Location:** `api-gateway/internal/grpc/`, `payment-service/internal/grpc/`

```go
// Every gRPC call uses context.WithTimeout
callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

resp, err := c.client.LockSeat(callCtx, request)
```

**Applied to:**
- ✅ `LockSeat` calls - 5 second timeout
- ✅ `ReleaseSeat` calls - 5 second timeout
- ✅ `BookSeat` calls - 5 second timeout
- ✅ `GetSeatStatus` calls - 5 second timeout
- ✅ Payment confirmation calls - 10 second timeout
- ✅ Client connection establishment - 10 second timeout

### 3. Buffered Channels for Concurrency

**Location:** `payment-service/internal/worker/worker.go`

```go
// Buffered channel with capacity of 1000 for flash sale spikes
var Queue = make(chan *PaymentQueueItem, 1000)

type WorkerPool struct {
    numWorkers      int
    processingQueue chan *PaymentQueueItem // Buffered for concurrent processing
    stopChan        chan struct{}
}
```

**Features:**
- ✅ 1000-capacity buffered channel for payment queue
- ✅ Multiple worker goroutines (configurable, default 10)
- ✅ Non-blocking queue submission
- ✅ Graceful shutdown with stop channel
- ✅ Each payment processed concurrently by different workers

### 4. Bi-directional gRPC Streaming

**Location:** `inventory-service/internal/handler/seat_handler.go`

```go
// Implements bi-directional streaming
func (h *SeatHandler) SeatStream(stream seatv1.SeatService_SeatStreamServer) error {
    // Receive from client in one goroutine
    // Send to client in another goroutine
    // Broadcast updates to all subscribers
}
```

**Features:**
- ✅ Real-time seat status updates
- ✅ Multiple concurrent subscribers
- ✅ Buffered event channels (10-capacity)
- ✅ Non-blocking broadcasts to avoid deadlocks

### 5. GraphQL Subscriptions

**Location:** `api-gateway/graph/schema.graphqls` and `resolver.go`

```graphql
type Subscription {
  # Subscribe to all seat updates
  seatUpdated: SeatEvent!
  
  # Subscribe to specific seat events
  seatEvents(seatId: ID!): SeatEvent!
}
```

## Data Flow

### Seat Locking Flow

1. **Client** → GraphQL `lockSeat` mutation
2. **API Gateway** → gRPC `LockSeat` to Inventory Service (with 5s timeout)
3. **Inventory Service:**
   - Locks seat in repository
   - Starts 5-minute expiry timer in goroutine
   - Broadcasts `SeatEvent` via streaming
4. **API Gateway** → gRPC `StartPaymentTimer` to Payment Service (with 10s timeout)
5. **Payment Service:**
   - Creates payment record
   - Starts payment timer in goroutine
6. **API Gateway** → GraphQL subscription sends updates to client

### Payment Confirmation Flow

1. **Client** → GraphQL `confirmPayment` mutation with transaction ID
2. **API Gateway** → gRPC `ConfirmPayment` to Payment Service (with 10s timeout)
3. **Payment Service:**
   - Validates and confirms payment
   - Cancels expiry timer
4. **API Gateway** → gRPC `BookSeat` to Inventory Service (with 5s timeout)
5. **Inventory Service:**
   - Confirms booking
   - Cancels lock timer
   - Broadcasts `SeatEvent`
6. **API Gateway** → GraphQL subscription notifies all clients

### Automatic Expiry Flow

1. **5-minute timeout expires** on seat lock
2. **Inventory Service** auto-calls `ReleaseSeat` via timer goroutine
3. **Payment Service** simultaneously times out payment if not confirmed
4. **Both services** broadcast update events
5. **API Gateway** sends GraphQL subscription updates
6. **Seat becomes available** for other users

## Error Handling

### Context Timeout Scenarios

- **gRPC call timeout:** Inventory service is slow/unresponsive
  - Payment is cancelled/refunded
  - Seat is released for other users
  
- **Payment timer timeout:** User doesn't complete payment in 5 minutes
  - Seat is auto-released
  - Payment is marked as expired

### Flash Sale Handling

- Buffered channel absorbs burst of lock requests
- 10 concurrent workers process payments
- System can handle 1000+ concurrent lock requests without crashing

## Testing

### Local Development

```bash
# Terminal 1: Start Inventory Service
cd inventory-service/cmd
go run main.go

# Terminal 2: Start Payment Service
cd payment-service/cmd
go run main.go

# Terminal 3: Start API Gateway
cd api-gateway/cmd
go run main.go

# Terminal 4: Access GraphQL Playground
# Open browser: http://localhost:8080
```

### GraphQL Query Examples

```graphql
# Lock a seat
mutation {
  lockSeat(seatId: "A1", userId: "user123") {
    status
    message
    expiresAtMs
  }
}

# Get seat status
query {
  seatStatus(seatId: "A1") {
    id
    status
    userId
    expiresAtMs
  }
}

# Confirm payment
mutation {
  confirmPayment(
    seatId: "A1"
    userId: "user123"
    transactionId: "txn_abc123"
  ) {
    status
    message
  }
}

# Subscribe to seat updates (real-time)
subscription {
  seatEvents(seatId: "A1") {
    seatId
    status
    userId
    timestampMs
  }
}
```

## Concurrency Guarantees

1. **Thread-Safe Repository:** Uses `sync.RWMutex` for all data access
2. **Goroutine Safety:** Each seat has its own timer goroutine
3. **Channel Safety:** Uses buffered channels with proper closure handling
4. **Context Cancellation:** Proper context passing through call chain
5. **Atomic Operations:** All state changes are atomic

## Timeout Configuration

| Operation | Timeout | Reason |
|-----------|---------|--------|
| Seat Lock | 5 min | User must complete payment |
| Payment Timer | 5 min | User must confirm payment |
| Lock/Release gRPC | 5 sec | Quick operations |
| Payment gRPC | 10 sec | Payment processing may take time |
| Client Connection | 10 sec | Service bootstrap |

## Performance Characteristics

- **Lock Operation:** O(1) - Direct map access
- **Release Operation:** O(1) - Direct map access
- **Broadcast:** O(n) - n = number of subscribers
- **Worker Processing:** Concurrent - 10 workers by default
- **Memory:** O(m) - m = number of active locks + payments

## Future Improvements

1. **Persistence:** Replace in-memory repository with database
2. **Distributed Tracing:** Add OpenTelemetry instrumentation
3. **Metrics:** Add Prometheus metrics for monitoring
4. **Redis Cache:** Use Redis for distributed locking
5. **Message Queue:** Replace buffered channels with Kafka/RabbitMQ
6. **Load Balancing:** Add load balancer for multiple instances
7. **Rate Limiting:** Implement rate limiting for API endpoints

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                   API Gateway (Port 8080)                    │
│  - GraphQL Server                                             │
│  - Context Timeouts on gRPC calls                             │
│  - GraphQL Subscriptions for real-time updates               │
└──────────┬──────────────────────────────┬──────────────────┘
           │                              │
    (gRPC with timeout)           (gRPC with timeout)
           │                              │
           ▼                              ▼
┌──────────────────────────────┐  ┌──────────────────────────────┐
│  Inventory Service (50051)   │  │  Payment Service (50052)      │
│  - Seat Locking              │  │  - Payment Timer Management   │
│  - Lock Expiry Timer         │  │  - Payment Processing         │
│  - Streaming Updates         │  │  - Worker Pool               │
│  - Thread-Safe Repo          │  │  - Buffered Channels         │
│  - Goroutine-based Timers    │  │  - Goroutine Timers          │
└──────────────────────────────┘  └──────────────────────────────┘
         │ Lock Update Events      │ Payment Status Events
         └────────────┬─────────────┘
                      ▼
            ┌──────────────────────┐
            │  Real-time Updates   │
            │  - WebSocket Streams │
            │  - GraphQL Subscriptions│
            └──────────────────────┘
```

## Key Files Modified/Created

### Protobuf
- `proto/seat/v1/seat.proto` - Updated with new message types
- `proto/seat/v1/seat.pb.go` - Generated message types
- `proto/seat/v1/seat_grpc.pb.go` - Generated service definitions

### Inventory Service
- `inventory-service/internal/model/seat.go` - Added expiry fields
- `inventory-service/internal/repository/seat_repo.go` - Added locking methods
- `inventory-service/internal/usecase/seat_usecase.go` - Implemented timer logic
- `inventory-service/internal/handler/seat_handler.go` - Implemented streaming

### Payment Service
- `payment-service/internal/model/payment.go` - New payment model
- `payment-service/internal/repository/payment_repo.go` - New repository
- `payment-service/internal/usecase/payment_usecase.go` - Payment logic
- `payment-service/internal/handler/payment_handler.go` - gRPC service
- `payment-service/internal/worker/worker.go` - Worker pool with buffered channels
- `payment-service/internal/timer/timer.go` - Timer management
- `payment-service/internal/grpc/server.go` - gRPC server setup

### API Gateway
- `api-gateway/graph/schema.graphqls` - Updated with mutations and subscriptions
- `api-gateway/graph/resolver.go` - Implemented resolvers
- `api-gateway/graph/model/models_gen.go` - Updated models
- `api-gateway/internal/grpc/inventory_client.go` - Added context timeouts
- `api-gateway/internal/grpc/payment_client.go` - Added context timeouts
- `api-gateway/cmd/main.go` - Updated initialization

---

**Implementation Date:** April 16, 2026  
**Status:** Complete with all architectural requirements implemented
