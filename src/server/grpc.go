package server

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/devlikeapro/noweb2/gows"
	pb "github.com/devlikeapro/noweb2/proto"
	"github.com/golang/protobuf/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/grpc"
	"log"
	"reflect"
	"strings"
	"time"
)

type Server struct {
	pb.UnimplementedMessageServiceServer
	pb.UnimplementedEventStreamServer
	EventChannel chan interface{}
	Gows         *gows.GoWS
}

func (s *Server) StartSession(ctx context.Context, req *pb.Session) (*pb.Empty, error) {
	log.Printf("Starting session...")
	err := s.Gows.Start()
	if !errors.Is(err, whatsmeow.ErrAlreadyConnected) {
		return nil, err
	}
	return &pb.Empty{}, nil
}

func (s *Server) StopSession(ctx context.Context, req *pb.Session) (*pb.Empty, error) {
	log.Printf("Stopping session...")
	s.Gows.Stop()
	return &pb.Empty{}, nil
}

func (s *Server) SendText(ctx context.Context, req *pb.TextMessageRequest) (*pb.MessageResponse, error) {
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}

	cli := s.Gows
	res, err := cli.SendMessage(context.Background(), jid, &waE2E.Message{
		Conversation: proto.String(req.GetText()),
	})

	if err != nil {
		return nil, err
	}

	return &pb.MessageResponse{Id: res.ID, Timestamp: time.Now().Unix()}, nil
}
func (s *Server) GetProfilePicture(ctx context.Context, req *pb.ProfilePictureRequest) (*pb.ProfilePictureResponse, error) {
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}

	cli := s.Gows
	info, err := cli.GetProfilePictureInfo(jid, &whatsmeow.GetProfilePictureParams{
		Preview: false,
	})
	if err != nil {
		return nil, err
	}

	return &pb.ProfilePictureResponse{Url: info.URL}, nil
}

func (s *Server) StreamEvents(req *pb.Empty, stream grpc.ServerStreamingServer[pb.EventJson]) error {
	for event := range s.EventChannel {
		jsonData, err := json.Marshal(event)
		if err != nil {
			continue
		}

		// Remove * at the start if it's *
		eventType := reflect.TypeOf(event).String()
		eventType = strings.TrimPrefix(eventType, "*")

		data := pb.EventJson{
			Event: reflect.TypeOf(event).String(),
			Data:  string(jsonData),
		}
		err = stream.Send(&data)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) IssueEvent(event interface{}) {
	s.EventChannel <- event
}
