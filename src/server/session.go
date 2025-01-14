package server

import (
	"context"
	"errors"
	"github.com/devlikeapro/gows/gows"
	"github.com/devlikeapro/gows/proto"
	"go.mau.fi/whatsmeow"
	"net/url"
)

func addApplicationName(address string, name string) string {
	parsedURL, err := url.Parse(address)
	if err != nil {
		return address
	}
	queryParams := parsedURL.Query()
	queryParams.Set("application_name", name)
	parsedURL.RawQuery = queryParams.Encode()
	return parsedURL.String()
}

func (s *Server) StartSession(ctx context.Context, req *__.StartSessionRequest) (*__.Empty, error) {
	dialect := req.Config.Store.Dialect
	var address string
	switch {
	case dialect == "sqlite3":
		address = req.Config.Store.Address + "?_foreign_keys=on"
	case dialect == "sqlite":
		address = req.Config.Store.Address + "?_foreign_keys=on"
	case dialect == "postgres":
		address = addApplicationName(req.Config.Store.Address, "GOWS")
	default:
		return nil, errors.New("unsupported sql dialect: " + dialect)
	}

	cfg := gows.SessionConfig{
		Store: gows.StoreConfig{
			Dialect: dialect,
			Address: address,
		},
		Log: gows.LogConfig{
			Level: req.Config.Log.Level.String(),
		},
		Proxy: gows.ProxyConfig{
			Url: req.Config.Proxy.Url,
		},
	}

	session := req.GetId()
	cli, err := s.Sm.Start(session, cfg)
	if err != nil {
		return nil, err
	}

	// Subscribe to events
	go func() {
		for evt := range cli.GetEventChannel() {
			s.IssueEvent(session, evt)
		}
	}()

	return &__.Empty{}, nil
}

func (s *Server) StopSession(ctx context.Context, req *__.Session) (*__.Empty, error) {
	s.Sm.Stop(req.GetId())
	return &__.Empty{}, nil
}

func (s *Server) GetSessionState(ctx context.Context, req *__.Session) (*__.SessionStateResponse, error) {
	cli, err := s.Sm.Get(req.GetId())
	if errors.Is(err, gows.ErrSessionNotFound) {
		return &__.SessionStateResponse{Found: false, Connected: false}, nil
	}
	if err != nil {
		return nil, err
	}
	return &__.SessionStateResponse{Found: true, Connected: cli.IsConnected()}, nil
}

func (s *Server) RequestCode(ctx context.Context, req *__.PairCodeRequest) (*__.PairCodeResponse, error) {
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
	return &__.PairCodeResponse{Code: code}, nil
}

func (s *Server) Logout(ctx context.Context, req *__.Session) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.GetId())
	if err != nil {
		return nil, err
	}
	err = cli.Logout()
	if err != nil {
		if errors.Is(err, whatsmeow.ErrNotLoggedIn) {
			// Ignore not logged in error
			return &__.Empty{}, nil
		}
		return nil, err
	}
	return &__.Empty{}, nil
}
