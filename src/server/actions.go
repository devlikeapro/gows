package server

import (
	"context"
	"errors"
	"github.com/devlikeapro/gows/media"
	"github.com/devlikeapro/gows/proto"
	"github.com/golang/protobuf/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

func (s *Server) SendMessage(ctx context.Context, req *__.MessageRequest) (*__.MessageResponse, error) {
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}

	message := waE2E.Message{}
	if req.Media == nil {
		message.Conversation = proto.String(req.Text)
	} else {
		var mediaType whatsmeow.MediaType
		switch req.Media.Type {
		case __.MediaType_IMAGE:
			// Upload
			mediaType = whatsmeow.MediaImage
			resp, err := cli.Upload(ctx, req.Media.Content, mediaType)
			if err != nil {
				return nil, err
			}
			// Generate Thumbnail
			thumbnail, err := media.ImageThumbnail(req.Media.Content)
			if err != nil {
				s.log.Errorf("Failed to generate thumbnail: %v", err)
			}
			// Attach
			message.ImageMessage = &waE2E.ImageMessage{
				Caption:       proto.String(req.Text),
				Mimetype:      proto.String(req.Media.Mimetype),
				JPEGThumbnail: thumbnail,
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
			}
		case __.MediaType_AUDIO:
			mediaType = whatsmeow.MediaAudio
			// Attach
			waveform, err := media.Waveform(req.Media.Content)
			if err != nil {
				s.log.Errorf("Failed to generate waveform: %v", err)
			}
			duration, err := media.Duration(req.Media.Content)
			if err != nil {
				s.log.Errorf("Failed to get duration of audio: %v", err)
			}
			durationSeconds := uint32(duration)

			// Upload
			resp, err := cli.Upload(ctx, req.Media.Content, mediaType)
			if err != nil {
				return nil, err
			}

			// Attach
			message.AudioMessage = &waE2E.AudioMessage{
				Mimetype:      proto.String(req.Media.Mimetype),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
				Seconds:       &durationSeconds,
				Waveform:      waveform,
			}
		case __.MediaType_VIDEO:
			mediaType = whatsmeow.MediaVideo
			// Upload
			resp, err := cli.Upload(ctx, req.Media.Content, mediaType)
			if err != nil {
				return nil, err
			}
			// Generate Thumbnail
			thumbnail, err := media.VideoThumbnail(
				req.Media.Content,
				0,
				struct{ Width int }{Width: 72},
			)

			if err != nil {
				s.log.Infof("Failed to generate video thumbnail: %v", err)
			}

			message.VideoMessage = &waE2E.VideoMessage{
				Caption:       proto.String(req.Text),
				Mimetype:      proto.String(req.Media.Mimetype),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
				JPEGThumbnail: thumbnail,
			}

		case __.MediaType_DOCUMENT:
			mediaType = whatsmeow.MediaDocument
			// Upload
			resp, err := cli.Upload(ctx, req.Media.Content, mediaType)
			if err != nil {
				return nil, err
			}

			// Generate Thumbnail if possible
			thumbnail, err := media.ImageThumbnail(req.Media.Content)
			if err != nil {
				s.log.Infof("Failed to generate thumbnail: %v", err)
			}

			// Attach
			message.DocumentMessage = &waE2E.DocumentMessage{
				Caption:       proto.String(req.Text),
				Mimetype:      proto.String(req.Media.Mimetype),
				URL:           &resp.URL,
				DirectPath:    &resp.DirectPath,
				MediaKey:      resp.MediaKey,
				FileEncSHA256: resp.FileEncSHA256,
				FileSHA256:    resp.FileSHA256,
				FileLength:    &resp.FileLength,
				JPEGThumbnail: thumbnail,
			}

		}

	}
	res, err := cli.SendMessage(context.Background(), jid, &message)
	if err != nil {
		return nil, err
	}

	return &__.MessageResponse{Id: res.ID, Timestamp: res.Timestamp.Unix()}, nil
}

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
