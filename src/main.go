package main

import (
	"context"
	"fmt"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"time"

	pb "github.com/devlikeapro/noweb2/proto"
)

type server struct {
	pb.UnimplementedMessageServiceServer
	client *whatsmeow.Client
}

func (s *server) SendText(ctx context.Context, req *pb.TextMessageRequest) (*pb.MessageResponse, error) {
	log.Printf("Received message: %s\n", req.Text)
	return &pb.MessageResponse{Id: "1", Timestamp: time.Now().Unix()}, nil
}

func BuildClient() *whatsmeow.Client {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				// e.g. qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
				fmt.Println("QR code:", evt.Code)
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}
	return client
}

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
	client := BuildClient()
	client.AddEventHandler(eventHandler)

	pb.RegisterMessageServiceServer(grpcServer, &server{
		client: client,
	})

	log.Println("Server is listening on port 50051")

	// Start the server
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
