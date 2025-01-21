package server

import (
	"context"
	"errors"
	"github.com/devlikeapro/gows/proto"
	"go.mau.fi/whatsmeow/types"
)

func (s *Server) SendPresence(ctx context.Context, req *__.PresenceRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}

	var presence types.Presence
	switch req.Status {
	case __.Presence_AVAILABLE:
		presence = types.PresenceAvailable
	case __.Presence_UNAVAILABLE:
		presence = types.PresenceUnavailable
	default:
		return nil, errors.New("invalid presence")
	}

	err = cli.SendPresence(presence)
	if err != nil {
		return nil, err
	}
	return &__.Empty{}, nil
}
func (s *Server) SendChatPresence(ctx context.Context, req *__.ChatPresenceRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	var presence types.ChatPresence
	var presenceMedia types.ChatPresenceMedia
	switch req.Status {
	case __.ChatPresence_TYPING:
		presence = types.ChatPresenceComposing
		presenceMedia = types.ChatPresenceMediaText
	case __.ChatPresence_RECORDING:
		presence = types.ChatPresenceComposing
		presenceMedia = types.ChatPresenceMediaAudio
	case __.ChatPresence_PAUSED:
		presence = types.ChatPresencePaused
		presenceMedia = types.ChatPresenceMediaText
	default:
		return nil, errors.New("invalid chat presence")
	}
	err = cli.SendChatPresence(jid, presence, presenceMedia)
	if err != nil {
		return nil, err
	}
	return &__.Empty{}, nil
}

func (s *Server) SubscribePresence(ctx context.Context, req *__.SubscribePresenceRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	err = cli.SendPresence(types.PresenceAvailable)
	if err != nil {
		return nil, err
	}
	err = cli.SubscribePresence(jid)
	if err != nil {
		return nil, err
	}
	return &__.Empty{}, nil
}
