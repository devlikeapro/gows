package server

import (
	"context"
	"encoding/json"
	"github.com/devlikeapro/noweb2/gows"
	pb "github.com/devlikeapro/noweb2/proto"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/grpc"
	"reflect"
	"strings"
	"sync"
	"time"
)

type Server struct {
	pb.UnimplementedMessageServiceServer
	pb.UnimplementedEventStreamServer
	Sm *gows.SessionManager

	listeners     map[uuid.UUID]chan interface{}
	listenersLock sync.RWMutex
}

func NewServer() *Server {
	return &Server{
		Sm:            gows.NewSessionManager(),
		listeners:     make(map[uuid.UUID]chan interface{}),
		listenersLock: sync.RWMutex{},
	}
}

func (s *Server) StartSession(ctx context.Context, req *pb.StartSessionRequest) (*pb.Empty, error) {
	dialect := req.Dialect
	address := req.Address + "?_foreign_keys=on"

	cli, err := s.Sm.Start(req.GetId(), dialect, address)
	if err != nil {
		return nil, err
	}

	// Subscribe to events
	go func() {
		for evt := range cli.Events {
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

func (s *Server) addListener(id uuid.UUID) chan interface{} {
	s.listenersLock.Lock()
	defer s.listenersLock.Unlock()

	listener := make(chan interface{}, 10)
	s.listeners[id] = listener
	return listener
}

func (s *Server) removeListener(id uuid.UUID) {
	s.listenersLock.Lock()
	defer s.listenersLock.Unlock()
	listener, ok := s.listeners[id]
	if !ok {
		return
	}
	delete(s.listeners, id)
	close(listener)
}
func (s *Server) getListeners() []chan interface{} {
	s.listenersLock.RLock()
	defer s.listenersLock.RUnlock()
	listeners := make([]chan interface{}, 0, len(s.listeners))
	for _, listener := range s.listeners {
		listeners = append(listeners, listener)
	}
	return listeners
}

func (s *Server) StreamEvents(req *pb.Empty, stream grpc.ServerStreamingServer[pb.EventJson]) error {
	streamId := uuid.New()
	listener := s.addListener(streamId)
	defer s.removeListener(streamId)

	for event := range listener {
		// Remove * at the start if it's *
		eventType := reflect.TypeOf(event).String()
		eventType = strings.TrimPrefix(eventType, "*")

		// TODO: Extract session name
		name := "default"

		jsonData, err := json.Marshal(event)
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
	listeners := s.getListeners()
	for _, listener := range listeners {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// Ignore panics
				}
			}()
			listener <- event
		}()
	}
}
