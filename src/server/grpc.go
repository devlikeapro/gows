package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devlikeapro/gows/gows"
	pb "github.com/devlikeapro/gows/proto"
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

	// session id -> id -> event channel
	listeners     map[string]map[uuid.UUID]chan interface{}
	listenersLock sync.RWMutex
}

func NewServer() *Server {
	return &Server{
		Sm:            gows.NewSessionManager(),
		listeners:     map[string]map[uuid.UUID]chan interface{}{},
		listenersLock: sync.RWMutex{},
	}
}

func (s *Server) StartSession(ctx context.Context, req *pb.StartSessionRequest) (*pb.Empty, error) {
	cfg := gows.SessionConfig{
		Store: gows.StoreConfig{
			Dialect: req.Config.Store.Dialect,
			Address: req.Config.Store.Address + "?_foreign_keys=on",
		},
		Log: gows.LogConfig{
			Level: req.Config.Log.Level.String(),
		},
	}

	session := req.GetId()
	cli, err := s.Sm.Start(session, cfg)
	if err != nil {
		return nil, err
	}

	// Subscribe to events
	go func() {
		for evt := range cli.Events {
			s.IssueEvent(session, evt)
		}
	}()

	return &pb.Empty{}, nil
}

func (s *Server) StopSession(ctx context.Context, req *pb.Session) (*pb.Empty, error) {
	s.Sm.Stop(req.GetId())
	return &pb.Empty{}, nil
}

func (s *Server) GetSessionState(ctx context.Context, req *pb.Session) (*pb.SessionStateResponse, error) {
	cli, err := s.Sm.Get(req.GetId())
	if errors.Is(err, gows.ErrSessionNotFound) {
		return &pb.SessionStateResponse{Found: false, Connected: false}, nil
	}
	if err != nil {
		return nil, err
	}
	return &pb.SessionStateResponse{Found: true, Connected: cli.IsConnected()}, nil
}

func (s *Server) RequestCode(ctx context.Context, req *pb.PairCodeRequest) (*pb.PairCodeResponse, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	code, err := cli.PairPhone(
		req.GetPhone(),
		true,
		whatsmeow.PairClientChrome,
		"Chrome (Linux)",
	)
	if err != nil {
		return nil, err
	}
	return &pb.PairCodeResponse{Code: code}, nil
}

func (s *Server) Logout(ctx context.Context, req *pb.Session) (*pb.Empty, error) {
	cli, err := s.Sm.Get(req.GetId())
	if err != nil {
		return nil, err
	}
	err = cli.Logout()
	if err != nil {
		if errors.Is(err, whatsmeow.ErrNotLoggedIn) {
			// Ignore not logged in error
			return &pb.Empty{}, nil
		}
		return nil, err
	}
	return &pb.Empty{}, nil
}

func (s *Server) SendMessage(ctx context.Context, req *pb.MessageRequest) (*pb.MessageResponse, error) {
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
	if errors.Is(err, whatsmeow.ErrProfilePictureNotSet) {
		return &pb.ProfilePictureResponse{Url: ""}, nil
	}
	if errors.Is(err, whatsmeow.ErrProfilePictureUnauthorized) {
		return &pb.ProfilePictureResponse{Url: ""}, nil
	}
	if err != nil {
		return nil, err
	}

	return &pb.ProfilePictureResponse{Url: info.URL}, nil
}

func (s *Server) addListener(session string, id uuid.UUID) chan interface{} {
	s.listenersLock.Lock()
	defer s.listenersLock.Unlock()

	listener := make(chan interface{}, 10)
	sessionListeners, ok := s.listeners[session]
	if !ok {
		sessionListeners = map[uuid.UUID]chan interface{}{}
		s.listeners[session] = sessionListeners
	}
	sessionListeners[id] = listener
	return listener
}

func (s *Server) removeListener(session string, id uuid.UUID) {
	s.listenersLock.Lock()
	defer s.listenersLock.Unlock()
	listener, ok := s.listeners[session][id]
	if !ok {
		return
	}
	delete(s.listeners[session], id)
	// if it's the last listener, remove the session
	if len(s.listeners[session]) == 0 {
		delete(s.listeners, session)
	}
	close(listener)
}

func (s *Server) getListeners(session string) []chan interface{} {
	s.listenersLock.RLock()
	defer s.listenersLock.RUnlock()
	listeners := make([]chan interface{}, 0, len(s.listeners))
	for _, listener := range s.listeners[session] {
		listeners = append(listeners, listener)
	}
	return listeners
}

func (s *Server) StreamEvents(req *pb.Session, stream grpc.ServerStreamingServer[pb.EventJson]) error {
	name := req.GetId()
	streamId := uuid.New()
	listener := s.addListener(name, streamId)
	defer s.removeListener(name, streamId)

	for event := range listener {
		// Remove * at the start if it's *
		eventType := reflect.TypeOf(event).String()
		eventType = strings.TrimPrefix(eventType, "*")

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

func (s *Server) IssueEvent(session string, event interface{}) {
	listeners := s.getListeners(session)
	for _, listener := range listeners {
		go func() {
			defer func() {
				if err := recover(); err != nil {
					// Print log error and ignore
					fmt.Print("Error when sending event to listener: ", err)
				}
			}()
			listener <- event
		}()
	}
}
