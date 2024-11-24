package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"os"
	"time"

	pb "github.com/devlikeapro/noweb2/proto/gen"
)

type server struct {
	pb.UnimplementedMessageServiceServer
}

// Sum implements the SumService.Sum RPC method.
func (s *server) Sum(ctx context.Context, req *pb.SumRequest) (*pb.SumResponse, error) {
	var result int64
	for _, num := range req.Numbers {
		result += num
	}
	if result == 10 {
		return nil, status.Errorf(codes.Code(13), "10 is evil")
	}
	return &pb.SumResponse{Result: result}, nil
}

func (s *server) StreamTimestamps(stream pb.TimestampService_StreamTimestampsServer) error {
	log.Println("Client connected to StreamTimestamps")

	// Start streaming timestamps to the client
	for {
		timestamp := time.Now().Unix()
		response := &pb.TimestampResponse{Timestamp: timestamp}

		// Send the current timestamp to the client
		if err := stream.Send(response); err != nil {
			log.Printf("Error sending timestamp: %v", err)
			return err
		}

		time.Sleep(1 * time.Second) // Wait for 1 second before sending the next timestamp
	}
}

func main() {
	socket := "/tmp/noweb2.sock"

	// Force remove the socket file
	_ = os.Remove(socket)
	// Listen on a specified port
	listener, err := net.Listen("unix", socket)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create a new gRPC server
	grpcServer := grpc.NewServer()

	// Register the TimestampService with the server
	pb.RegisterSumServiceServer(grpcServer, &server{})
	pb.RegisterTimestampServiceServer(grpcServer, &server{})

	log.Println("Server is listening on port 50051")

	// Start the server
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
