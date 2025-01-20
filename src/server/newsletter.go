package server

import (
	"context"
	"errors"
	"github.com/devlikeapro/gows/gows"
	__ "github.com/devlikeapro/gows/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
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

func (s *Server) GetNewsletterInfo(ctx context.Context, req *__.NewsletterInfoRequest) (*__.Newsletter, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	id := req.GetId()
	if gows.HasNewsletterSuffix(id) {
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

func (s *Server) CreateNewsletter(ctx context.Context, req *__.CreateNewsletterRequest) (*__.Newsletter, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	params := whatsmeow.CreateNewsletterParams{
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Picture:     req.GetPicture(),
	}
	resp, err := cli.CreateNewsletter(params)
	if err != nil {
		return nil, err
	}
	return toNewsletter(resp), nil

}

func (s *Server) NewsletterToggleMute(ctx context.Context, req *__.NewsletterToggleMuteRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	if !gows.IsNewsletter(jid) {
		return nil, errors.New("invalid jid, not a newsletter")
	}
	err = cli.NewsletterToggleMute(jid, req.GetMute())
	if err != nil {
		return nil, err
	}
	return &__.Empty{}, nil
}

// NewsletterToggleFollow
func (s *Server) NewsletterToggleFollow(ctx context.Context, req *__.NewsletterToggleFollowRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.GetSession().GetId())
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.GetJid())
	if err != nil {
		return nil, err
	}
	if !gows.IsNewsletter(jid) {
		return nil, errors.New("invalid jid, not a newsletter")
	}
	if req.Follow {
		err = cli.FollowNewsletter(jid)
	} else {
		err = cli.UnfollowNewsletter(jid)
	}
	if err != nil {
		return nil, err
	}
	return &__.Empty{}, nil
}
