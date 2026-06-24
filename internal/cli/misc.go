package cli

import (
	"github.com/alecthomas/kong"

	"github.com/rnwolfe/knit/internal/errs"
	"github.com/rnwolfe/knit/internal/skill"
	"github.com/rnwolfe/knit/internal/version"
)

// --- auth -------------------------------------------------------------------
// PLACEHOLDER. cli-implement wires the Threads OAuth model (spec.md §Auth): --token-stdin
// (primary, headless) + paste-the-callback-URL (human onboarding), long-lived 60d token with
// refresh, secrets in the OS keyring + 0600 fallback (contract §7). Secrets via stdin/env,
// NEVER argv. Nothing here authenticates yet.

type AuthCmd struct {
	Status  AuthStatusCmd  `cmd:"" help:"Test the token; report scopes, expiry, and advanced-access state."`
	Login   AuthLoginCmd   `cmd:"" help:"Authenticate (paste-the-callback-URL, or --token-stdin)."`
	Logout  AuthLogoutCmd  `cmd:"" help:"Remove stored credentials from the keyring/file."`
	Refresh AuthRefreshCmd `cmd:"" help:"Extend the long-lived (60-day) token; safe on a schedule."`
}

type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(rt *Runtime) error {
	// cli-implement: load token from keyring, validate, fill these fields + remaining publish quota.
	return rt.Out.Emit(map[string]any{
		"authenticated":  false,
		"method":         "oauth",
		"scopes":         []string{},
		"expiresInDays":  nil,
		"advancedSearch": false,
		"note":           "placeholder — run `knit auth login` once auth is wired (cli-implement)",
	})
}

// AuthLoginCmd: --token-stdin reads a pre-provisioned token from stdin (never argv). With no
// flag, cli-implement prints the authorize URL and reads the pasted callback URL/code on stdin.
type AuthLoginCmd struct {
	TokenStdin bool `name:"token-stdin" help:"Read a pre-obtained token from stdin (headless; no browser)."`
}

func (c *AuthLoginCmd) Run(rt *Runtime) error {
	rt.Out.Info("auth is not wired yet (cli-implement). See spec.md §Auth for the token-stdin and paste-the-callback-URL flows.")
	return rt.Out.Emit(map[string]any{"ok": false, "method": "oauth", "tokenStdin": c.TokenStdin})
}

type AuthLogoutCmd struct{}

func (c *AuthLogoutCmd) Run(rt *Runtime) error {
	return rt.Out.Emit(map[string]any{"ok": true})
}

type AuthRefreshCmd struct{}

func (c *AuthRefreshCmd) Run(rt *Runtime) error {
	// cli-implement: exchange the unexpired long-lived token for a fresh 60-day token.
	rt.Out.Info("token refresh is not wired yet (cli-implement)")
	return rt.Out.Emit(map[string]any{"ok": false, "note": "placeholder"})
}

// --- doctor -----------------------------------------------------------------

type DoctorCmd struct {
	ForAgent bool `name:"for-agent" help:"Terser, machine-skimmable diagnostics for an agent."`
}

func (c *DoctorCmd) Run(rt *Runtime) error {
	// PLACEHOLDER checks. cli-implement: verify KNIT_CLIENT_ID/SECRET, keyring reachability,
	// token validity/expiry, connectivity to graph.threads.net, and granted scopes.
	checks := []map[string]any{
		{"name": "config", "ok": true, "detail": "KNIT_CLIENT_ID/SECRET check not wired yet"},
		{"name": "keyring", "ok": true, "detail": "keyring check not wired yet"},
		{"name": "auth", "ok": true, "detail": "token check not wired yet"},
		{"name": "connectivity", "ok": true, "detail": "graph.threads.net reachability not wired yet"},
	}
	allOK := true
	for _, ch := range checks {
		if ok, _ := ch["ok"].(bool); !ok {
			allOK = false
		}
	}
	if !allOK {
		return errs.New(errs.ExitConfig, "DOCTOR_FAILED", "one or more checks failed", "see the failing check's detail")
	}
	if c.ForAgent {
		return rt.Out.Emit(map[string]any{"ok": true, "checks": len(checks)})
	}
	return rt.Out.Emit(map[string]any{"ok": true, "checks": checks})
}

// --- schema -----------------------------------------------------------------

type SchemaCmd struct{}

func (c *SchemaCmd) Run(rt *Runtime) error {
	k, err := kong.New(&CLI{}, kong.Name("knit"))
	if err != nil {
		return errs.New(errs.ExitGeneric, "SCHEMA_ERROR", err.Error(), "")
	}
	out := map[string]any{
		"tool":       "knit",
		"version":    version.String(),
		"commands":   nodeToMap(k.Model.Node),
		"exit_codes": errs.Table(),
		"safety": map[string]any{
			"allow_mutations": rt.Cfg.AllowMutations,
			"dry_run":         rt.Cfg.DryRun,
			"no_input":        rt.Cfg.NoInput,
		},
	}
	return rt.Out.EmitJSON(out) // schema is always JSON
}

func nodeToMap(n *kong.Node) map[string]any {
	m := map[string]any{"name": n.Name}
	if n.Help != "" {
		m["help"] = n.Help
	}
	var flags []map[string]any
	for _, f := range n.Flags {
		if f.Name == "help" {
			continue
		}
		fm := map[string]any{"name": f.Name}
		if f.Help != "" {
			fm["help"] = f.Help
		}
		if f.Default != "" {
			fm["default"] = f.Default
		}
		flags = append(flags, fm)
	}
	if len(flags) > 0 {
		m["flags"] = flags
	}
	var args []map[string]any
	for _, p := range n.Positional {
		args = append(args, map[string]any{"name": p.Name, "help": p.Help})
	}
	if len(args) > 0 {
		m["args"] = args
	}
	var subs []any
	for _, ch := range n.Children {
		subs = append(subs, nodeToMap(ch))
	}
	if len(subs) > 0 {
		m["subcommands"] = subs
	}
	return m
}

// --- agent ------------------------------------------------------------------

type AgentCmd struct{}

func (c *AgentCmd) Run(rt *Runtime) error {
	_, err := rt.Out.Stdout.Write([]byte(skill.Content))
	return err
}

// --- version ----------------------------------------------------------------

type VersionCmd struct{}

func (c *VersionCmd) Run(rt *Runtime) error {
	return rt.Out.Emit(map[string]any{"version": version.String()})
}
