package main

import (
	"flag"
	pb "github.com/devlikeapro/noweb2/proto"
	"github.com/devlikeapro/noweb2/server"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
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

func buildGrpcServer() *grpc.Server {
	grpcServer := grpc.NewServer()
	srv := server.NewServer()
	// Add an event handler to the client
	pb.RegisterMessageServiceServer(grpcServer, srv)
	pb.RegisterEventStreamServer(grpcServer, srv)
	return grpcServer
}

var socket string

func init() {
	flag.StringVar(&socket, "socket", "/tmp/gows.sock", "Socket path")
}

func main() {
	flag.Parse()
	// Build the server
	grpcServer := buildGrpcServer()
	// Open unix socket
	log.Println("Opening socket", socket)
	listener := listenSocket(socket)

	// Start the server
	log.Println("gRPC server started!")
	if err := grpcServer.Serve(*listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
