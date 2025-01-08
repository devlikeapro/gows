package server

import (
	"encoding/json"
	"fmt"
	"github.com/devlikeapro/gows/proto"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"reflect"
	"strings"
)

func (s *Server) safeMarshal(v interface{}) (result string) {
	defer func() {
		if err := recover(); err != nil {
			// Print log error and ignore
			s.log.Errorf("Error when marshaling data: %v", err)
			result = ""
		}
	}()
	data, err := json.Marshal(v)
	if err != nil {
		s.log.Errorf("Error when marshaling data: %v", err)
		return ""
	}
	result = string(data)
	return result
}

func (s *Server) StreamEvents(req *__.Session, stream grpc.ServerStreamingServer[__.EventJson]) error {
	name := req.GetId()
	streamId := uuid.New()
	listener := s.addListener(name, streamId)
	defer s.removeListener(name, streamId)
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case event := <-listener:
			// Remove * at the start if it's *
			eventType := reflect.TypeOf(event).String()
			eventType = strings.TrimPrefix(eventType, "*")

			jsonString := s.safeMarshal(event)
			if jsonString == "" {
				continue
			}

			data := __.EventJson{
				Session: name,
				Event:   eventType,
				Data:    jsonString,
			}
			err := stream.Send(&data)
			if err != nil {
				return err
			}
		}
	}
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
