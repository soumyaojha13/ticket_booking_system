# Ticket Booking System

A microservices-based ticket booking system built with Go, featuring real-time seat management, payment processing, and GraphQL API.

## Features

- **Real-time Seat Management**: Bidirectional gRPC streaming for live seat updates
- **5-Minute Seat Locking**: Automatic expiry mechanism using goroutines and timers
- **Payment Processing**: Worker pool with buffered channels for handling high concurrency
- **GraphQL API**: Subscriptions for real-time frontend updates
- **Context Timeouts**: Prevents hanging on gRPC calls
- **Multi-Database Support**: MongoDB, PostgreSQL, and Redis integration

## Architecture

The system consists of three main services:

### 1. Inventory Service (Port 50051)
- Manages seat availability and locking
- Broadcasts seat status updates via bidirectional streaming
- Uses hybrid repository pattern (MongoDB + Redis)

### 2. Payment Service (Port 50052)
- Handles payment timers and processing
- Worker pool for concurrent payment handling
- Supports MongoDB and PostgreSQL repositories

### 3. API Gateway (Port 8080)
- GraphQL endpoint for client interactions
- Manages communication between services
- Provides real-time subscriptions

## Prerequisites

- Go 1.26.1 or later
- Protocol Buffers compiler (protoc)
- Docker and Docker Compose (for containerized deployment)
- MongoDB, PostgreSQL, Redis (or use Docker)

## Quick Start

### Using Docker Compose

1. Clone the repository:
   ```bash
   git clone https://github.com/soumyaojha13/ticket_booking_system.git
   cd ticket_booking_system
   ```

2. Start all services:
   ```bash
   docker-compose up --build
   ```

### Manual Setup

1. Install dependencies:
   ```bash
   go mod download
   ```

2. Generate protobuf files:
   ```bash
   protoc --go_out=. --go-grpc_out=. proto/seat/v1/seat.proto
   ```

3. Start services in order:

   **Inventory Service:**
   ```bash
   cd inventory-service/cmd
   go run main.go
   ```

   **Payment Service:**
   ```bash
   cd payment-service/cmd
   go run main.go
   ```

   **API Gateway:**
   ```bash
   cd api-gateway/cmd
   go run main.go
   ```

4. Access GraphQL playground at `http://localhost:8080`

## Testing

Run tests for individual services:

```bash
# Inventory Service
cd inventory-service
go test ./...

# Payment Service
cd payment-service
go test ./...
```

## API Documentation

- GraphQL endpoint: `http://localhost:8080`
- Inventory gRPC: `localhost:50051`
- Payment gRPC: `localhost:50052`

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License.