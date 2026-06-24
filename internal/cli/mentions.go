package cli

import "github.com/rnwolfe/knit/internal/api"

// MentionsCmd lists public posts mentioning the authenticated user. Free text from other
// users → fenced in agent mode (contract §8).
type MentionsCmd struct {
	List MentionsListCmd `cmd:"" help:"List public posts that mention you."`
}

type MentionsListCmd struct {
	Cursor string `help:"Opaque pagination cursor."`
}

func (c *MentionsListCmd) Run(rt *Runtime) error {
	posts, cursor, err := rt.API.Mentions(rt.Ctx, api.PageOpts{Limit: rt.Cfg.Limit, Cursor: c.Cursor})
	if err != nil {
		return err
	}
	rt.fencePosts(posts)
	return rt.Out.EmitEnvelope(posts, cursor)
}
