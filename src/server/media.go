package server

import (
	"context"
	"encoding/json"
	"github.com/devlikeapro/gows/proto"
	"go.mau.fi/whatsmeow/proto/waE2E"
)

func (s *Server) DownloadMedia(ctx context.Context, req *__.DownloadMediaRequest) (*__.DownloadMediaResponse, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	msg, err := BuildMessage(req.GetMessage())
	if err != nil {
		return nil, err
	}
	resp, err := cli.DownloadAny(msg)
	if err != nil {
		s.log.Errorf("Failed to download media: %v", err)
		return nil, nil
	}
	return &__.DownloadMediaResponse{Content: resp}, nil
}

// BuildMessage builds a message from the given JSON data
func BuildMessage(data string) (*waE2E.Message, error) {
	var message waE2E.Message
	err := json.Unmarshal([]byte(data), &message)
	if err != nil {
		return nil, err
	}
	return &message, nil
}
