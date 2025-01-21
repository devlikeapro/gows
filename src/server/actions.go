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
	"time"
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
		var backgroundArgb *uint32
		if req.BackgroundColor != nil {
			backgroundArgb, err = media.ParseColor(req.BackgroundColor.Value)
			if err != nil {
				return nil, err
			}
		}

		var font *waE2E.ExtendedTextMessage_FontType
		if req.Font != nil {
			font = media.ParseFont(req.Font.Value)
		}

		message.ExtendedTextMessage = &waE2E.ExtendedTextMessage{
			Text:           proto.String(req.Text),
			BackgroundArgb: backgroundArgb,
			Font:           font,
		}
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

func (s *Server) MarkRead(ctx context.Context, req *__.MarkReadRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.Jid)
	if err != nil {
		return nil, err
	}

	sender, err := types.ParseJID(req.Sender)
	if err != nil {
		return nil, err
	}

	var receiptType types.ReceiptType
	switch req.Type {
	case __.ReceiptType_READ:
		receiptType = types.ReceiptTypeRead
	case __.ReceiptType_PLAYED:
		receiptType = types.ReceiptTypePlayed
	default:
		return nil, errors.New("invalid receipt type: " + req.Type.String())
	}

	// id to ids array
	ids := []types.MessageID{req.MessageId}
	now := time.Now()
	err = cli.MarkRead(ids, now, jid, sender, receiptType)
	if err != nil {
		return nil, err
	}
	return &__.Empty{}, nil
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
