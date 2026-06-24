package cli

import "github.com/rnwolfe/knit/internal/api"

// InsightsCmd reads post- and account-level metrics. Numeric → no injection fencing needed.
type InsightsCmd struct {
	Post    InsightsPostCmd    `cmd:"" help:"Per-post metrics by media id."`
	Account InsightsAccountCmd `cmd:"" help:"Account-level metrics + demographics."`
}

type InsightsPostCmd struct {
	ID string `arg:"" help:"Media id of the post."`
}

func (c *InsightsPostCmd) Run(rt *Runtime) error {
	metrics, err := rt.API.PostInsights(rt.Ctx, c.ID)
	if err != nil {
		return err
	}
	return rt.Out.EmitEnvelope(metrics, "")
}

type InsightsAccountCmd struct {
	Metrics []string `help:"Subset of metrics to return (default: all)."`
	Since   string   `help:"ISO-8601 start of the window."`
	Until   string   `help:"ISO-8601 end of the window."`
}

func (c *InsightsAccountCmd) Run(rt *Runtime) error {
	metrics, err := rt.API.AccountInsights(rt.Ctx, api.AccountInsightsOpts{
		Metrics: c.Metrics, Since: c.Since, Until: c.Until,
	})
	if err != nil {
		return err
	}
	return rt.Out.EmitEnvelope(metrics, "")
}
