package cli

// SearchCmd searches public Threads posts. Results are attacker-controllable free text →
// fence as untrusted in agent mode (contract §8). Without advanced access the API silently
// scopes to the user's own posts, so the envelope carries scope:public|self (spec.md).
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
	// PLACEHOLDER. cli-implement wires `GET /keyword_search` and sets scope from granted access.
	return rt.Out.EmitEnvelopeWith([]any{}, "", map[string]any{"scope": "self"})
}
