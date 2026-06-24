package cli

// ProfileCmd reads a Threads profile. Placeholder: cli-implement wires `GET /me` / `GET /{user-id}`.
type ProfileCmd struct {
	Get ProfileGetCmd `cmd:"" help:"Get a profile (self or by user id)."`
}

type ProfileGetCmd struct {
	User string `arg:"" optional:"" default:"me" help:"User id, or 'me' (default)."`
}

func (c *ProfileGetCmd) Run(rt *Runtime) error {
	// PLACEHOLDER profile shaped per spec.md (biography is user-controlled → fence in agent mode).
	profile := map[string]any{
		"id":                c.User,
		"username":          "",
		"name":              "",
		"biography":         "",
		"profilePictureUrl": "",
	}
	return rt.Out.EmitEnvelope(profile, "")
}
