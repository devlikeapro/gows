package gows

import (
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
	log.Printf("Starting session '%s'...", name)
	sm.sessionsLock.Lock()
	defer sm.sessionsLock.Unlock()
	if goWS, ok := sm.sessions[name]; ok {
		return goWS, nil
	}
	gows := BuildSession(dialect, address)
	err := gows.Start()
	if err != nil && !errors.Is(err, whatsmeow.ErrAlreadyConnected) {
		return nil, err
	}
	sm.sessions[name] = gows
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
	log.Printf("Stopping session '%s'...", name)
	sm.sessionsLock.Lock()
	defer sm.sessionsLock.Unlock()

	if goWS, ok := sm.sessions[name]; ok {
		goWS.Stop()
		delete(sm.sessions, name)
	}
	log.Printf("Session stopped '%s'", name)
}
