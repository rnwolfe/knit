package cli

// ProfileCmd reads a Threads profile. The biography is user-controlled → fenced in agent mode.
type ProfileCmd struct {
	Get ProfileGetCmd `cmd:"" help:"Get a profile (self or by user id)."`
}

type ProfileGetCmd struct {
	User string `arg:"" optional:"" default:"me" help:"User id, or 'me' (default)."`
}

func (c *ProfileGetCmd) Run(rt *Runtime) error {
	p, err := rt.API.Profile(rt.Ctx, c.User)
	if err != nil {
		return err
	}
	rt.fenceProfile(p)
	return rt.Out.EmitEnvelope(p, "")
}
