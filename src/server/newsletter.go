package server

import (
	"context"
	__ "github.com/devlikeapro/gows/proto"
	"go.mau.fi/whatsmeow/types"
	"strings"
)

func toNewsletter(n *types.NewsletterMetadata) *__.Newsletter {
	var picture string
	if n.ThreadMeta.Picture != nil {
		picture = n.ThreadMeta.Picture.URL
		if picture == "" {
			picture = n.ThreadMeta.Picture.DirectPath
		}
	}

	var preview string
	preview = n.ThreadMeta.Preview.URL
	if preview == "" {
		preview = n.ThreadMeta.Preview.DirectPath
	}
	var role string
	if n.ViewerMeta != nil {
		role = string(n.ViewerMeta.Role)
	}
	return &__.Newsletter{
		Id:          n.ID.String(),
		Name:        n.ThreadMeta.Name.Text,
		Description: n.ThreadMeta.Description.Text,
		Invite:      n.ThreadMeta.InviteCode,
		Picture:     picture,
		Preview:     preview,
		Verified:    n.ThreadMeta.VerificationState == types.NewsletterVerificationStateVerified,
		Role:        role,
	}
}

func (s *Server) GetSubscribedNewsletters(ctx context.Context, req *__.NewsletterListRequest) (*__.NewsletterList, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	resp, err := cli.GetSubscribedNewsletters()
	if err != nil {
		return nil, err
	}
	list := make([]*__.Newsletter, len(resp))
	for i, n := range resp {
		picture := n.ThreadMeta.Picture.URL
		if picture == "" {
			picture = n.ThreadMeta.Picture.DirectPath
		}
		preview := n.ThreadMeta.Preview.URL
		if preview == "" {
			preview = n.ThreadMeta.Preview.DirectPath
		}
		list[i] = toNewsletter(n)
	}
	return &__.NewsletterList{Newsletters: list}, nil
}

const newsletterJIDSuffix = "@newsletter"

func (s *Server) GetNewsletterInfo(ctx context.Context, req *__.NewsletterInfoRequest) (*__.Newsletter, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	id := req.GetId()
	if strings.HasSuffix(id, newsletterJIDSuffix) {
		jid, err := types.ParseJID(id)
		if err != nil {
			return nil, err
		}
		resp, err := cli.GetNewsletterInfo(jid)
		if err != nil {
			return nil, err
		}
		return toNewsletter(resp), nil
	}
	resp, err := cli.GetNewsletterInfoWithInvite(id)
	if err != nil {
		return nil, err
	}
	return toNewsletter(resp), nil
}
