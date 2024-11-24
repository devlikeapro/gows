package main

import (
	"fmt"
	pb "github.com/devlikeapro/noweb2/proto"
	"github.com/devlikeapro/noweb2/server"
	"github.com/devlikeapro/noweb2/service"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"

	_ "github.com/mattn/go-sqlite3" // Import the SQLite drive
)

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message!", v.Message.GetConversation())
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
	client := service.BuildClient()
	client.AddEventHandler(eventHandler)

	pb.RegisterMessageServiceServer(grpcServer, &server.Server{
		Client: client,
	})

	log.Println("Server is listening on", socket)

	// Start the server
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
