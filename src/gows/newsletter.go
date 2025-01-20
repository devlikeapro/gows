package gows

import (
	"go.mau.fi/whatsmeow/types"
	"strings"
)

func HasNewsletterSuffix(s string) bool {
	return strings.HasSuffix(s, "@"+types.NewsletterServer)
}

func IsNewsletter(jid types.JID) bool {
	return jid.Server == types.NewsletterServer
}
