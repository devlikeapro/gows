package gows

import (
	"context"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func (gows *GoWS) UploadMedia(
	ctx context.Context,
	jid types.JID,
	content []byte,
	mediaType whatsmeow.MediaType,
) (resp whatsmeow.UploadResponse, err error) {
	if IsNewsletter(jid) {
		resp, err = gows.UploadNewsletter(ctx, content, mediaType)
	} else {
		resp, err = gows.Upload(ctx, content, mediaType)
	}
	return resp, err
}
