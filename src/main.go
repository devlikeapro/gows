package main

import (
	"flag"
	gowsLog "github.com/devlikeapro/gows/log"
	pb "github.com/devlikeapro/gows/proto"
	"github.com/devlikeapro/gows/server"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/grpc"
	"net"
	"os"
)

func listenSocket(log waLog.Logger, path string) *net.Listener {
	log.Infof("Server is listening on %s", path)
	// Force remove the socket file
	_ = os.Remove(path)
	// Listen on a specified port
	listener, err := net.Listen("unix", path)
	if err != nil {
		log.Errorf("Failed to listen: %v", err)
	}
	return &listener
}

func buildGrpcServer() *grpc.Server {
	// 128 MB
	maxMessageSize := 128 * 1024 * 1024

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxMessageSize),
		grpc.MaxSendMsgSize(maxMessageSize),
	)
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

func remove(path string) {
	_ = os.Remove(path)
}

func main() {
	flag.Parse()
	log := gowsLog.Stdout("Server", "DEBUG", false)
	// Build the server
	grpcServer := buildGrpcServer()
	// Open unix socket
	log.Infof("Opening socket %s", socket)
	listener := listenSocket(log, socket)
	defer remove(socket)

	// Start the server
	log.Infof("gRPC server started!")
	if err := grpcServer.Serve(*listener); err != nil {
		log.Errorf("Failed to serve: %v", err)
	}
}
