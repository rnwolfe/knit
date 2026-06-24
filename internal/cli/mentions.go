package cli

// MentionsCmd lists public posts mentioning the authenticated user. Free text from other
// users → fence as untrusted in agent mode (contract §8). Advanced-access gated like search.
type MentionsCmd struct {
	List MentionsListCmd `cmd:"" help:"List public posts that mention you."`
}

type MentionsListCmd struct {
	Cursor string `help:"Opaque pagination cursor."`
}

func (c *MentionsListCmd) Run(rt *Runtime) error {
	// PLACEHOLDER. cli-implement wires `GET /me/mentions`.
	return rt.Out.EmitEnvelopeWith([]any{}, "", map[string]any{"scope": "self"})
}
