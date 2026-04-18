package grpc

import (
	"log"
	"net"
	"os"

	"github.com/soumyaojha/ticket-booking-system/inventory-service/internal/handler"
	seatv1 "github.com/soumyaojha/ticket-booking-system/proto/seat/v1"

	"google.golang.org/grpc"
)

func StartServer(h *handler.SeatHandler) {
	port := os.Getenv("INVENTORY_PORT")
	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()

	seatv1.RegisterSeatServiceServer(s, h)

	log.Printf("Inventory Service running on :%s\n", port)

	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
