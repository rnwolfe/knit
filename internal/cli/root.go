// Package cli wires the kong grammar, the runtime, and the exit-code mapping.
// main() does nothing but os.Exit(cli.Run(...)) so every path is testable in-process.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/rnwolfe/knit/internal/errs"
	"github.com/rnwolfe/knit/internal/output"
	"github.com/rnwolfe/knit/internal/store"
)

// CLI is the kong grammar. Global flags are the universal agent-CLI contract surface;
// subcommands follow noun-verb grammar.
type CLI struct {
	// Output (contract §1, §6)
	Format   string `enum:"json,plain,tsv" default:"plain" help:"Output format: json, plain, or tsv."`
	JSON     bool   `help:"Shorthand for --format=json."`
	NoColor  bool   `help:"Disable colored output."`
	Limit    int    `default:"50" help:"Maximum items to return for list operations."`
	Select   string `help:"Comma-separated dot-path field projection, e.g. id,title."`
	Concise  bool   `help:"Terser output (default)."`
	Detailed bool   `help:"Richer output."`

	// Safety (contract §2)
	AllowMutations bool `help:"Permit state-changing operations (off by default)."`
	DryRun         bool `help:"Print intended mutations without performing them."`
	Yes            bool `help:"Assume yes for confirmations (scripting)."`
	Force          bool `help:"Bypass safety checks."`
	NoInput        bool `help:"Never prompt; fail with exit 13 instead."`

	// Commands (noun-verb, service-namespaced)
	Profile  ProfileCmd  `cmd:"" help:"Read Threads profiles."`
	Post     PostCmd     `cmd:"" help:"List, read, and publish posts."`
	Reply    ReplyCmd    `cmd:"" help:"Read replies and manage replies on your posts."`
	Search   SearchCmd   `cmd:"" help:"Search public posts by keyword or topic."`
	Mentions MentionsCmd `cmd:"" help:"List public posts mentioning you."`
	Insights InsightsCmd `cmd:"" help:"Read post- and account-level metrics."`
	Auth     AuthCmd     `cmd:"" help:"Manage authentication."`
	Doctor   DoctorCmd   `cmd:"" help:"Diagnose setup and report fixes."`
	Schema   SchemaCmd   `cmd:"" help:"Print the machine-readable command schema (JSON)."`
	Agent    AgentCmd    `cmd:"" help:"Print the bundled agent SKILL.md."`
	Version  VersionCmd  `cmd:"" help:"Print the version."`
}

// Runtime is the per-invocation context bound into every command's Run method.
type Runtime struct {
	Cfg   *CLI
	Out   *output.Writer
	Store *store.Store
	Stdin io.Reader
}

// Guard enforces the read-only-by-default mutation gate (contract §2).
func (rt *Runtime) Guard(op string) error {
	if rt.Cfg.AllowMutations {
		return nil
	}
	return errs.MutationBlocked(op)
}

// Run parses args and dispatches, returning the process exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var cfg CLI
	helpShown := false
	parser, err := kong.New(&cfg,
		kong.Name("knit"),
		kong.Description("An agent-friendly CLI for Instagram's Threads. Read-only by default; mutations require --allow-mutations."),
		kong.Writers(stdout, stderr),
		kong.Exit(func(int) { helpShown = true }), // --help/--version: we control exit
	)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		return errs.ExitGeneric
	}

	kctx, perr := parser.Parse(args)
	if helpShown {
		return errs.ExitOK
	}
	if perr != nil {
		return handleParseError(stderr, args, perr)
	}

	if cfg.JSON {
		cfg.Format = "json"
	}
	rt := newRuntime(&cfg, stdin, stdout, stderr)

	if err := kctx.Run(rt); err != nil {
		return emitError(rt, err)
	}
	return errs.ExitOK
}

func newRuntime(cfg *CLI, stdin io.Reader, stdout, stderr io.Writer) *Runtime {
	format := output.Format(cfg.Format)
	color := !cfg.NoColor && os.Getenv("NO_COLOR") == "" && isTTY(stdout) && format == output.FormatPlain
	var sel []string
	if cfg.Select != "" {
		sel = strings.Split(cfg.Select, ",")
	}
	w := &output.Writer{
		Stdout: stdout, Stderr: stderr,
		Format: format, Color: color, Limit: cfg.Limit, Select: sel,
	}
	return &Runtime{Cfg: cfg, Out: w, Store: store.New(store.DefaultPath()), Stdin: stdin}
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// emitError prints a structured error to stderr and returns its exit code (contract §3).
func emitError(rt *Runtime, err error) int {
	var ce *errs.CLIError
	if !errors.As(err, &ce) {
		ce = errs.New(errs.ExitGeneric, "INTERNAL", err.Error(), "")
	}
	if rt.Out.Format == output.FormatJSON {
		enc := json.NewEncoder(rt.Out.Stderr)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{
			"error":       ce.Message,
			"code":        ce.Code,
			"remediation": ce.Remediation,
		})
	} else {
		fmt.Fprintf(rt.Out.Stderr, "error: %s\n", ce.Message)
		if ce.Code != "" {
			fmt.Fprintf(rt.Out.Stderr, "  code: %s\n", ce.Code)
		}
		if ce.Remediation != "" {
			fmt.Fprintf(rt.Out.Stderr, "  fix:  %s\n", ce.Remediation)
		}
	}
	return ce.Exit
}

// handleParseError reports usage errors and offers a "did you mean" suggestion.
func handleParseError(stderr io.Writer, args []string, err error) int {
	fmt.Fprintf(stderr, "error: %s\n", err)
	commands := []string{"profile", "post", "reply", "search", "mentions", "insights", "auth", "doctor", "schema", "agent", "version"}
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		if s, ok := closest(a, commands); ok {
			fmt.Fprintf(stderr, "  did you mean %q?\n", s)
		}
		break
	}
	return errs.ExitUsage
}
