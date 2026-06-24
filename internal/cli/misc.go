package cli

import (
	"github.com/alecthomas/kong"

	"github.com/rnwolfe/knit/internal/errs"
	"github.com/rnwolfe/knit/internal/skill"
	"github.com/rnwolfe/knit/internal/version"
)

// auth and doctor live in authcmd.go and doctor.go.

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
