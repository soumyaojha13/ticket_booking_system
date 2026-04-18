// cmd/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/soumyaojha/ticket-booking-system/api-gateway/graph"
	"github.com/soumyaojha/ticket-booking-system/api-gateway/internal/grpc"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	log.Println("API Gateway starting...")

	// Create gRPC clients with retry logic
	var inventoryClient *grpc.InventoryClient
	var paymentClient *grpc.PaymentClient

	// Retry connecting to services
	for i := 0; i < 5; i++ {
		if i > 0 {
			time.Sleep(2 * time.Second)
		}

		c, err := grpc.NewInventoryClient()
		if err != nil {
			log.Printf("Inventory connect attempt %d failed: %v\n", i+1, err)
			continue
		}
		inventoryClient = c
		log.Println("Connected to Inventory Service")
		break
	}

	for i := 0; i < 5; i++ {
		if i > 0 {
			time.Sleep(2 * time.Second)
		}

		c, err := grpc.NewPaymentClient()
		if err != nil {
			log.Printf("Payment connect attempt %d failed: %v\n", i+1, err)
			continue
		}
		paymentClient = c
		log.Println("Connected to Payment Service")
		break
	}

	if inventoryClient == nil || paymentClient == nil {
		log.Fatal("Failed to connect to required services")
	}

	// Create resolver with gRPC clients
	resolver := &graph.Resolver{
		Inventory: inventoryClient,
		Payment:   paymentClient,
	}

	// Bridge Inventory gRPC seat stream -> GraphQL subscriptions.
	streamCtx, streamCancel := context.WithCancel(context.Background())
	defer streamCancel()
	go resolver.StartSeatStream(streamCtx)

	// Generate executable schema
	config := graph.Config{Resolvers: resolver}
	schema := graph.NewExecutableSchema(config)

	// Create GraphQL handler
	srv := handler.NewDefaultServer(schema)

	// Setup HTTP routes
	http.Handle("/", playground.Handler("GraphQL Playground", "/query"))
	http.Handle("/query", srv)

	// Add health check endpoint
	http.HandleFunc("/health", healthCheck)

	log.Println("GraphQL running at http://localhost:8080")
	log.Println("GraphQL Playground at http://localhost:8080/")

	httpPort := os.Getenv("API_GATEWAY_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	server := &http.Server{
		Addr:    ":" + httpPort,
		Handler: nil,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		streamCancel()
		_ = inventoryClient.Close()
		_ = paymentClient.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}

// healthCheck provides a simple health check endpoint
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}
