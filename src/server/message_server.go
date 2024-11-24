package server

import (
	"context"
	"github.com/devlikeapro/noweb2/proto"
	"github.com/golang/protobuf/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"time"
)

type Server struct {
	__.UnimplementedMessageServiceServer
	Client *whatsmeow.Client
}

func (s *Server) SendText(ctx context.Context, req *__.TextMessageRequest) (*__.MessageResponse, error) {
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}

	cli := s.Client
	res, err := cli.SendMessage(context.Background(), jid, &waE2E.Message{
		Conversation: proto.String(req.GetText()),
	})

	if err != nil {
		return nil, err
	}

	return &__.MessageResponse{Id: res.ID, Timestamp: time.Now().Unix()}, nil
}
