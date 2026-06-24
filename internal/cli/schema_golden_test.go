package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSchemaSnapshot is the contract CI gate (contract §10): the machine-readable schema —
// command tree, flags, and exit-code table — must not change without a reviewed golden diff.
// Run `KNIT_UPDATE_GOLDEN=1 go test ./internal/cli/` to update after an intentional change.
func TestSchemaSnapshot(t *testing.T) {
	withFake(t)
	var out, errb bytes.Buffer
	if code := Run([]string{"schema"}, strings.NewReader(""), &out, &errb); code != 0 {
		t.Fatalf("schema exit = %d: %s", code, errb.String())
	}

	// Normalize: drop the volatile version field, re-marshal stably.
	var doc map[string]any
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("schema not JSON: %v", err)
	}
	delete(doc, "version")
	got, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')

	golden := filepath.Join("testdata", "schema.golden.json")
	if os.Getenv("KNIT_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run KNIT_UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("schema drifted from golden. If intentional, run:\n  KNIT_UPDATE_GOLDEN=1 go test ./internal/cli/\n\n--- got ---\n%s", got)
	}
}
