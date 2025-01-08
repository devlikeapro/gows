package gows

import (
	"context"
	"errors"
	gowsLog "github.com/devlikeapro/gows/log"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store"
	waLog "go.mau.fi/whatsmeow/util/log"
	"sync"
)

var ErrSessionNotFound = errors.New("session not found")

// SessionManager control sessions in thread-safe way
type SessionManager struct {
	sessions     map[string]*GoWS
	sessionsLock *sync.RWMutex
	log          waLog.Logger
}

type StoreConfig struct {
	Dialect string
	Address string
}

type LogConfig struct {
	Level string
}

type ProxyConfig struct {
	Url string
}

type SessionConfig struct {
	Store StoreConfig
	Log   LogConfig
	Proxy ProxyConfig
}

func init() {
	// Firefox (Ubuntu)
	store.DeviceProps.PlatformType = proto.DeviceProps_FIREFOX.Enum()
	store.SetOSInfo("Ubuntu", [3]uint32{22, 0, 4})
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:     make(map[string]*GoWS),
		sessionsLock: &sync.RWMutex{},
		log:          gowsLog.Stdout("Manager", "DEBUG", false),
	}
}

func (sm *SessionManager) Start(name string, cfg SessionConfig) (*GoWS, error) {
	sm.sessionsLock.Lock()
	defer sm.sessionsLock.Unlock()
	gows, err := sm.unlockedStart(name, cfg)
	if err != nil {
		sm.log.Errorf("Error starting session '%s': %v", name, err)
		sm.unlockedStop(name)
		return nil, err
	}
	return gows, nil
}

func (sm *SessionManager) unlockedStart(name string, cfg SessionConfig) (*GoWS, error) {
	sm.log.Infof("Starting session '%s'...", name)
	if goWS, ok := sm.sessions[name]; ok {
		return goWS, nil
	}

	ctx := context.WithValue(context.Background(), "name", name)
	log := gowsLog.Stdout("Session", cfg.Log.Level, false)

	dialect := cfg.Store.Dialect
	address := cfg.Store.Address
	gows, err := BuildSession(ctx, log.Sub(name), dialect, address)
	if err != nil {
		return nil, err
	}
	sm.sessions[name] = gows

	err = gows.SetProxyAddress(cfg.Proxy.Url)
	if err != nil {
		return nil, err
	}
	err = gows.Start()
	if err != nil && !errors.Is(err, whatsmeow.ErrAlreadyConnected) {
		return nil, err
	}
	sm.log.Infof("Session started '%s'", name)
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
	sm.log.Infof("Stopping session '%s'...", name)
	if goWS, ok := sm.sessions[name]; ok {
		goWS.Stop()
		delete(sm.sessions, name)
	}
	sm.log.Infof("Session stopped '%s'", name)
}
