package main

import (
	"flag"
	gows2 "github.com/devlikeapro/noweb2/gows"
	pb "github.com/devlikeapro/noweb2/proto"
	"github.com/devlikeapro/noweb2/server"
	"github.com/devlikeapro/noweb2/service"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"

	_ "github.com/mattn/go-sqlite3" // Import the SQLite drive
)

func listenSocket(path string) *net.Listener {
	log.Println("Server is listening on", path)
	// Force remove the socket file
	_ = os.Remove(path)
	// Listen on a specified port
	listener, err := net.Listen("unix", path)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	return &listener
}

func buildGrpcServer(gows *gows2.GoWS) *grpc.Server {
	grpcServer := grpc.NewServer()
	srv := server.Server{
		Gows:         gows,
		EventChannel: make(chan interface{}, 100),
	}

	// Add an event handler to the client
	pb.RegisterMessageServiceServer(grpcServer, &srv)
	pb.RegisterEventStreamServer(grpcServer, &srv)
	gows.AddEventHandler(srv.IssueEvent)
	// Subscribe to QrChan events
	go func() {
		for evt := range gows.QrChan {
			srv.IssueEvent(evt)
		}
	}()
	return grpcServer
}

var socket string

func init() {
	flag.StringVar(&socket, "socket", "/tmp/gows.sock", "Socket path")
}

func main() {
	flag.Parse()
	// Start the gows session
	gows := service.BuildSession()
	// Build the server
	grpcServer := buildGrpcServer(gows)
	// Open unix socket
	log.Println("Opening socket", socket)
	listener := listenSocket(socket)

	// Start the server
	log.Println("Starting gRPC server...")
	if err := grpcServer.Serve(*listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
