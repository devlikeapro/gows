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
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}

	message := waE2E.Message{}
	mediaResponse := whatsmeow.UploadResponse{}

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
			mediaResponse, err = cli.UploadMedia(ctx, jid, req.Media.Content, mediaType)
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
				URL:           &mediaResponse.URL,
				DirectPath:    &mediaResponse.DirectPath,
				FileSHA256:    mediaResponse.FileSHA256,
				FileLength:    &mediaResponse.FileLength,
				MediaKey:      mediaResponse.MediaKey,
				FileEncSHA256: mediaResponse.FileEncSHA256,
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
			mediaResponse, err = cli.UploadMedia(ctx, jid, req.Media.Content, mediaType)
			if err != nil {
				return nil, err
			}

			// Attach
			ptt := true
			message.AudioMessage = &waE2E.AudioMessage{
				Mimetype:      proto.String(req.Media.Mimetype),
				URL:           &mediaResponse.URL,
				DirectPath:    &mediaResponse.DirectPath,
				MediaKey:      mediaResponse.MediaKey,
				FileEncSHA256: mediaResponse.FileEncSHA256,
				FileSHA256:    mediaResponse.FileSHA256,
				FileLength:    &mediaResponse.FileLength,
				Seconds:       &durationSeconds,
				Waveform:      waveform,
				PTT:           &ptt,
			}
		case __.MediaType_VIDEO:
			mediaType = whatsmeow.MediaVideo
			// Upload
			mediaResponse, err = cli.UploadMedia(ctx, jid, req.Media.Content, mediaType)
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
				URL:           &mediaResponse.URL,
				DirectPath:    &mediaResponse.DirectPath,
				MediaKey:      mediaResponse.MediaKey,
				FileEncSHA256: mediaResponse.FileEncSHA256,
				FileSHA256:    mediaResponse.FileSHA256,
				FileLength:    &mediaResponse.FileLength,
				JPEGThumbnail: thumbnail,
			}

		case __.MediaType_DOCUMENT:
			mediaType = whatsmeow.MediaDocument
			// Upload
			mediaResponse, err = cli.UploadMedia(ctx, jid, req.Media.Content, mediaType)
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
				URL:           &mediaResponse.URL,
				DirectPath:    &mediaResponse.DirectPath,
				MediaKey:      mediaResponse.MediaKey,
				FileEncSHA256: mediaResponse.FileEncSHA256,
				FileSHA256:    mediaResponse.FileSHA256,
				FileLength:    &mediaResponse.FileLength,
				JPEGThumbnail: thumbnail,
			}
		}
	}

	extra := whatsmeow.SendRequestExtra{}
	if mediaResponse.Handle != "" {
		// Newsletters
		extra.MediaHandle = mediaResponse.Handle
	}

	res, err := cli.SendMessage(ctx, jid, &message, extra)
	if err != nil {
		return nil, err
	}

	return &__.MessageResponse{Id: res.ID, Timestamp: res.Timestamp.Unix()}, nil
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
