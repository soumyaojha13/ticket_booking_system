# Quick Start Guide

## Prerequisites

- Go 1.26.1 or later
- Protocol Buffers compiler (protoc)
- Docker (optional, for testing)

## Setup & Run

### 1. Start Inventory Service

```bash
cd inventory-service/cmd
go run main.go
```

Expected output:
```
Inventory Service running on :50051
```

### 2. Start Payment Service

```bash
cd payment-service/cmd
go run main.go
```

Expected output:
```
Payment Service starting on :50052
```

### 3. Start API Gateway

```bash
cd api-gateway/cmd
go run main.go
```

Expected output:
```
API Gateway starting...
Connected to Inventory Service
Connected to Payment Service
GraphQL running at http://localhost:8080
```

## Testing the System

### Option 1: GraphQL Playground

1. Open browser: `http://localhost:8080`
2. Try this mutation to lock a seat:

```graphql
mutation {
  lockSeat(seatId: "A1", userId: "user123") {
    status
    message
    seatId
    expiresAtMs
  }
}
```

3. Check seat status:

```graphql
query {
  seatStatus(seatId: "A1") {
    id
    status
    userId
    expiresAtMs
  }
}
```

4. Subscribe to real-time updates:

```graphql
subscription {
  seatEvents(seatId: "A1") {
    seatId
    status
    userId
    timestampMs
  }
}
```

5. Confirm payment (within 5 minutes):

```graphql
mutation {
  confirmPayment(seatId: "A1", userId: "user123", transactionId: "txn_123") {
    status
    message
  }
}
```

### Option 2: cURL

```bash
# Lock a seat
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "mutation { lockSeat(seatId: \"A1\", userId: \"user123\") { status message } }"}'

# Get status
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "query { seatStatus(seatId: \"A1\") { id status userId } }"}'

# Release seat
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "mutation { releaseSeat(seatId: \"A1\") }"}'
```

## Key Behaviors to Test

### 1. 5-Minute Lock Expiry
- Lock a seat
- Wait 10 seconds (or modify timeout in code to 10 seconds for faster testing)
- Status should automatically change to AVAILABLE

### 2. Concurrent Locks
- In multiple GraphQL tabs, lock different seats simultaneously
- Verify all locks are created correctly
- Verify broadcasts work (use subscriptions)

### 3. Payment Timeout
- Lock a seat (starts 5-minute payment timer)
- Don't confirm payment
- After 5 minutes, payment should expire
- Seat should become available

### 4. Successful Booking
- Lock seat A1
- Check status (should be LOCKED)
- Confirm payment
- Check status (should be BOOKED)

### 5. Release Locked Seat
- Lock seat B1
- Call releaseSeat mutation
- Verify status is AVAILABLE

## Code Structure

```
ticket-booking-system/
├── proto/
│   └── seat/v1/
│       ├── seat.proto
│       ├── seat.pb.go
│       └── seat_grpc.pb.go
├── inventory-service/
│   ├── cmd/main.go
│   └── internal/
│       ├── model/seat.go
│       ├── repository/seat_repo.go
│       ├── usecase/seat_usecase.go
│       ├── handler/seat_handler.go
│       └── grpc/server.go
├── payment-service/
│   ├── cmd/main.go
│   └── internal/
│       ├── model/payment.go
│       ├── repository/payment_repo.go
│       ├── usecase/payment_usecase.go
│       ├── handler/payment_handler.go
│       ├── worker/worker.go
│       ├── timer/timer.go
│       └── grpc/server.go
└── api-gateway/
    ├── cmd/main.go
    ├── graph/
    │   ├── resolver.go
    │   ├── schema.graphqls
    │   └── model/models_gen.go
    └── internal/grpc/
        ├── inventory_client.go
        └── payment_client.go
```

## Configuration

### Timeouts (seconds)
- Lock Duration: 5 minutes (300 seconds)
- Payment Timeout: 5 minutes (300 seconds)
- gRPC Call Timeout: 5 seconds
- Payment gRPC Timeout: 10 seconds

### Worker Pool
- Number of Workers: 10 (configurable)
- Queue Buffer Size: 1000
- Event Channel Buffer: 10

## Troubleshooting

### Services won't connect
```
Error: Failed to connect to inventory service
```
**Solution:** Make sure Inventory Service is running on port 50051

### GraphQL query fails
```
Error: Error getting seat status
```
**Solution:** Check if the seat ID exists or service is running

### Subscription not working
```
Subscription "seatEvents" does not have a subscription handler
```
**Solution:** Ensure gqlgen has generated subscription code. May need to regenerate:
```bash
cd api-gateway
go run github.com/99designs/gqlgen generate
```

## Performance Testing

### Test 1: Load Test
```bash
# Create 100 concurrent lock requests
for i in {1..100}; do
  curl -X POST http://localhost:8080/query \
    -H "Content-Type: application/json" \
    -d '{"query": "mutation { lockSeat(seatId: \"Seat'$i'\", userId: \"user'$i'\") { status } }"}' &
done
wait
```

### Test 2: Timer Test
```bash
# Lock a seat and wait for expiry (modify timer to 10s for testing)
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "mutation { lockSeat(seatId: \"A1\", userId: \"user1\") { status } }"}'

# Wait 10+ seconds, then check
sleep 11
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query": "query { seatStatus(seatId: \"A1\") { status } }"}'
# Should show AVAILABLE
```

## Debugging

### Enable logging
Set `GODEBUG` environment variable:
```bash
export GODEBUG=http2debug=1
go run main.go
```

### Inspect gRPC messages
Add logging in handler methods to see request/response

### Check goroutines
```go
import "runtime"
// In your code
fmt.Printf("Goroutines: %d\n", runtime.NumGoroutine())
```

## Next Steps

1. Add Redis for distributed locking in multi-instance setup
2. Add authentication/authorization
3. Add database persistence (PostgreSQL/MongoDB)
4. Add monitoring (Prometheus/Grafana)
5. Add tracing (Jaeger/OpenTelemetry)
6. Deploy with Docker/Kubernetes
