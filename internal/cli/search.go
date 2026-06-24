package cli

import "github.com/rnwolfe/knit/internal/api"

// SearchCmd searches public Threads posts. Results are attacker-controllable free text →
// fenced in agent mode (contract §8). The envelope carries scope:public|self so an agent knows
// whether advanced access was actually in effect (without it, the API returns only own posts).
type SearchCmd struct {
	Posts SearchPostsCmd `cmd:"" help:"Search posts by keyword or topic tag."`
}

type SearchPostsCmd struct {
	Keyword   string `arg:"" help:"Keyword or topic to search."`
	MediaType string `enum:",text,image,video" default:"" help:"Filter by media type."`
	Mode      string `enum:"keyword,tag" default:"keyword" help:"Search mode: keyword or topic tag."`
	Cursor    string `help:"Opaque pagination cursor."`
}

func (c *SearchPostsCmd) Run(rt *Runtime) error {
	posts, cursor, scope, err := rt.API.Search(rt.Ctx, api.SearchOpts{
		Query:     c.Keyword,
		MediaType: c.MediaType,
		Tag:       c.Mode == "tag",
		PageOpts:  api.PageOpts{Limit: rt.Cfg.Limit, Cursor: c.Cursor},
	})
	if err != nil {
		return err
	}
	if scope == "self" {
		rt.Out.Info("note: scope=self — results limited to your own posts (advanced access for public search not granted)")
	}
	rt.fencePosts(posts)
	return rt.Out.EmitEnvelopeWith(posts, cursor, map[string]any{"scope": scope})
}
