package server

import (
	"context"
	"errors"
	"github.com/devlikeapro/gows/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func (s *Server) GetProfilePicture(ctx context.Context, req *__.ProfilePictureRequest) (*__.ProfilePictureResponse, error) {
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
		return &__.ProfilePictureResponse{Url: ""}, nil
	}
	if errors.Is(err, whatsmeow.ErrProfilePictureUnauthorized) {
		return &__.ProfilePictureResponse{Url: ""}, nil
	}
	if err != nil {
		return nil, err
	}

	return &__.ProfilePictureResponse{Url: info.URL}, nil
}

func (s *Server) CheckPhones(ctx context.Context, req *__.CheckPhonesRequest) (*__.CheckPhonesResponse, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}

	phones := make([]string, len(req.Phones))
	for i, p := range req.Phones {
		// start with +
		if p[0] != '+' {
			p = "+" + p
		}
		phones[i] = p
	}

	res, err := cli.IsOnWhatsApp(phones)
	if err != nil {
		return nil, err
	}

	infos := make([]*__.PhoneInfo, len(res))
	for i, r := range res {
		infos[i] = &__.PhoneInfo{
			Phone:      r.Query,
			Jid:        r.JID.String(),
			Registered: r.IsIn,
		}
	}
	return &__.CheckPhonesResponse{Infos: infos}, nil
}
