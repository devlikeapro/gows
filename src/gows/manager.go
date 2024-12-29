package gows

import (
	"context"
	"errors"
	"go.mau.fi/whatsmeow"
	"log"
	"sync"
)

var ErrSessionNotFound = errors.New("session not found")

// SessionManager control sessions in thread-safe way
type SessionManager struct {
	sessions     map[string]*GoWS
	sessionsLock *sync.RWMutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:     make(map[string]*GoWS),
		sessionsLock: &sync.RWMutex{},
	}
}

func (sm *SessionManager) Start(name string, dialect string, address string) (*GoWS, error) {
	sm.sessionsLock.Lock()
	defer sm.sessionsLock.Unlock()
	gows, err := sm.unlockedStart(name, dialect, address)
	if err != nil {
		log.Printf("Error starting session '%s': %v", name, err)
		sm.unlockedStop(name)
		return nil, err
	}
	return gows, nil
}

func (sm *SessionManager) unlockedStart(name string, dialect string, address string) (*GoWS, error) {
	log.Printf("Starting session '%s'...", name)
	if goWS, ok := sm.sessions[name]; ok {
		return goWS, nil
	}
	ctx := context.WithValue(context.Background(), "name", name)
	gows, err := BuildSession(ctx, dialect, address)
	if err != nil {
		return nil, err
	}
	sm.sessions[name] = gows
	err = gows.Start()
	if err != nil && !errors.Is(err, whatsmeow.ErrAlreadyConnected) {
		return nil, err
	}
	log.Printf("Session started '%s'", name)
	return gows, nil
}

func (sm *SessionManager) Get(name string) (*GoWS, error) {
	sm.sessionsLock.RLock()
	defer sm.sessionsLock.RUnlock()

	if goWS, ok := sm.sessions[name]; !ok {
		return nil, ErrSessionNotFound
	} else {
		return goWS, nil
	}
}

func (sm *SessionManager) Stop(name string) {
	sm.sessionsLock.Lock()
	defer sm.sessionsLock.Unlock()
	sm.unlockedStop(name)
}

func (sm *SessionManager) unlockedStop(name string) {
	log.Printf("Stopping session '%s'...", name)
	if goWS, ok := sm.sessions[name]; ok {
		goWS.Stop()
		delete(sm.sessions, name)
	}
	log.Printf("Session stopped '%s'", name)
}
