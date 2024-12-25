package server

import (
	"context"
	"encoding/json"
	"github.com/devlikeapro/noweb2/gows"
	pb "github.com/devlikeapro/noweb2/proto"
	"github.com/golang/protobuf/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/grpc"
	"reflect"
	"strings"
	"time"
)

type Server struct {
	pb.UnimplementedMessageServiceServer
	pb.UnimplementedEventStreamServer
	EventChannel chan interface{}
	Sm           *gows.SessionManager
}

func (s *Server) StartSession(ctx context.Context, req *pb.StartSessionRequest) (*pb.Empty, error) {
	dialect := req.Dialect
	address := req.Address + "?_foreign_keys=on"

	cli, err := s.Sm.Start(req.GetId(), dialect, address)
	if err != nil {
		return nil, err
	}

	// Subscribe to events
	cli.AddEventHandler(s.IssueEvent)

	// Subscribe to QrChan events
	go func() {
		for evt := range cli.QrChan {
			s.IssueEvent(evt)
		}
	}()

	return &pb.Empty{}, nil
}

func (s *Server) StopSession(ctx context.Context, req *pb.Session) (*pb.Empty, error) {
	s.Sm.Stop(req.GetId())
	return &pb.Empty{}, nil
}

func (s *Server) SendText(ctx context.Context, req *pb.TextMessageRequest) (*pb.MessageResponse, error) {
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}

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

	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
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
		// Remove * at the start if it's *
		eventType := reflect.TypeOf(event).String()
		eventType = strings.TrimPrefix(eventType, "*")

		//TODO: Extract session
		name := "default"
		cli, err := s.Sm.Get(name)
		if err != nil {
			continue
		}

		var eventData interface{}
		switch event.(type) {
		case *events.Connected:
			eventData = &gows.ConnectedEventData{
				ID:       cli.Store.ID,
				PushName: cli.Store.PushName,
			}

		default:
			eventData = event
		}

		jsonData, err := json.Marshal(eventData)
		if err != nil {
			continue
		}

		data := pb.EventJson{
			Session: name,
			Event:   eventType,
			Data:    string(jsonData),
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
