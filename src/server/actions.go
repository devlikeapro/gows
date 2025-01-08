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
			var waveform []byte
			var duration float32
			// Get waveform and duration if available
			if req.Media.Audio != nil {
				waveform = req.Media.Audio.Waveform
				duration = req.Media.Audio.Duration
			}

			if waveform == nil || len(waveform) == 0 {
				// Generate waveform
				waveform, err = media.Waveform(req.Media.Content)
				if err != nil {
					s.log.Errorf("Failed to generate waveform: %v", err)
				}
			}
			if duration == 0 {
				// Get duration
				duration, err = media.Duration(req.Media.Content)
				if err != nil {
					s.log.Errorf("Failed to get duration of audio: %v", err)
				}
			}
			durationSeconds := uint32(duration)

			// Upload
			resp, err := cli.Upload(ctx, req.Media.Content, mediaType)
			if err != nil {
				return nil, err
			}

			// Attach
			ptt := true
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
				PTT:           &ptt,
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
	res, err := cli.SendMessage(ctx, jid, &message)
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

func (s *Server) SendReaction(ctx context.Context, req *__.MessageReaction) (*__.MessageResponse, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.Jid)
	sender, err := types.ParseJID(req.Sender)

	message := cli.BuildReaction(jid, sender, req.MessageId, req.Reaction)
	res, err := cli.SendMessage(ctx, jid, message)
	if err != nil {
		return nil, err
	}

	return &__.MessageResponse{Id: res.ID, Timestamp: res.Timestamp.Unix()}, nil
}
