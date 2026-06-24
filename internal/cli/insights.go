package cli

// InsightsCmd reads post- and account-level metrics. Numeric → no injection fencing needed.
type InsightsCmd struct {
	Post    InsightsPostCmd    `cmd:"" help:"Per-post metrics by media id."`
	Account InsightsAccountCmd `cmd:"" help:"Account-level metrics + demographics."`
}

type InsightsPostCmd struct {
	ID string `arg:"" help:"Media id of the post."`
}

func (c *InsightsPostCmd) Run(rt *Runtime) error {
	// PLACEHOLDER. cli-implement wires `GET /{media-id}/insights`.
	metrics := map[string]any{
		"views": nil, "likes": nil, "replies": nil,
		"reposts": nil, "quotes": nil, "shares": nil,
	}
	return rt.Out.EmitEnvelope(metrics, "")
}

type InsightsAccountCmd struct {
	Metrics []string `help:"Subset of metrics to return (default: all)."`
	Since   string   `help:"ISO-8601 start of the window."`
	Until   string   `help:"ISO-8601 end of the window."`
}

func (c *InsightsAccountCmd) Run(rt *Runtime) error {
	// PLACEHOLDER. cli-implement wires `GET /{user-id}/threads_insights`.
	metrics := map[string]any{
		"views": nil, "likes": nil, "replies": nil, "reposts": nil,
		"quotes": nil, "shares": nil, "followersCount": nil, "followerDemographics": nil,
	}
	return rt.Out.EmitEnvelope(metrics, "")
}
