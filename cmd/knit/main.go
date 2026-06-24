// Command knit is an agent-friendly CLI for Instagram's Threads (official Threads API).
//
// It is read-only by default with an in-binary mutation gate, structured errors + exit codes,
// a machine-readable `schema`, and an embedded agent SKILL.md. main() does nothing but
// os.Exit(cli.Run(...)) so every code path is testable in-process. See spec.md and AGENTS.md.
package main

import (
	"os"

	"github.com/rnwolfe/knit/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
