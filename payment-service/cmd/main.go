package main

import (
	"log"
	"os"
	"strconv"

	"github.com/soumyaojha/ticket-booking-system/payment-service/internal/grpc"
)

func main() {
	log.Println("Payment Service starting...")

	port := os.Getenv("PAYMENT_PORT")
	if port == "" {
		port = "50052"
	}

	workers := 10
	if v := os.Getenv("PAYMENT_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			workers = n
		}
	}

	grpc.StartPaymentServer(port, workers)
}
