package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(args, strings.NewReader(""), &out, &errb)
	return out.String(), errb.String(), code
}

func useTempStore(t *testing.T) {
	t.Helper()
	t.Setenv("KNIT_STORE", filepath.Join(t.TempDir(), "store.json"))
	t.Setenv("NO_COLOR", "1")
}

// Reads return the stable {schemaVersion, data, nextCursor?} envelope (spec.md).
func TestPostListEmptyEnvelope(t *testing.T) {
	useTempStore(t)
	out, _, code := run(t, "post", "list", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	var env struct {
		SchemaVersion int   `json:"schemaVersion"`
		Data          []any `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("stdout not valid JSON: %v\n%s", err, out)
	}
	if env.SchemaVersion != 1 {
		t.Fatalf("schemaVersion = %d, want 1", env.SchemaVersion)
	}
	if len(env.Data) != 0 {
		t.Fatalf("want empty data, got %v", env.Data)
	}
}

func TestMutationBlockedByDefault(t *testing.T) {
	useTempStore(t)
	out, errb, code := run(t, "post", "create", "--text", "hello", "--json")
	if code != 12 {
		t.Fatalf("exit = %d, want 12 (MUTATION_BLOCKED)", code)
	}
	if !strings.Contains(errb, "MUTATION_BLOCKED") {
		t.Fatalf("stderr missing MUTATION_BLOCKED: %s", errb)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("stdout should be empty on error, got: %s", out)
	}
}

func TestMutationAllowed(t *testing.T) {
	useTempStore(t)
	if _, _, code := run(t, "post", "create", "--text", "hello", "--allow-mutations", "--json"); code != 0 {
		t.Fatalf("create exit = %d, want 0", code)
	}
	out, _, code := run(t, "post", "list", "--json")
	if code != 0 {
		t.Fatalf("list exit = %d, want 0", code)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("created post not listed: %s", out)
	}
}

func TestDryRunChangesNothingAndHashes(t *testing.T) {
	useTempStore(t)
	out, _, code := run(t, "post", "create", "--text", "ghost", "--allow-mutations", "--dry-run", "--json")
	if code != 0 {
		t.Fatalf("dry-run exit = %d, want 0", code)
	}
	var plan map[string]any
	if err := json.Unmarshal([]byte(out), &plan); err != nil {
		t.Fatalf("dry-run output not JSON: %v", err)
	}
	if plan["dryRun"] != true || plan["hash"] == nil {
		t.Fatalf("dry-run missing dryRun/hash: %s", out)
	}
	listOut, _, _ := run(t, "post", "list", "--json")
	if strings.Contains(listOut, "ghost") {
		t.Fatalf("dry-run should not persist: %s", listOut)
	}
}

// --apply with a stale/wrong hash must refuse to publish (reviewed-artifact = approval).
func TestApplyHashMismatchRefuses(t *testing.T) {
	useTempStore(t)
	_, errb, code := run(t, "post", "create", "--text", "x", "--apply", "deadbeef", "--allow-mutations", "--json")
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (usage/PLAN_MISMATCH)", code)
	}
	if !strings.Contains(errb, "PLAN_MISMATCH") {
		t.Fatalf("stderr missing PLAN_MISMATCH: %s", errb)
	}
}

func TestSchemaHasSafetyAndExitCodes(t *testing.T) {
	useTempStore(t)
	out, _, code := run(t, "schema")
	if code != 0 {
		t.Fatalf("schema exit = %d, want 0", code)
	}
	var s map[string]any
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("schema not valid JSON: %v", err)
	}
	if _, ok := s["safety"]; !ok {
		t.Fatalf("schema missing safety state")
	}
	if _, ok := s["exit_codes"]; !ok {
		t.Fatalf("schema missing exit_codes")
	}
	// The full noun surface must be present in the schema.
	if !strings.Contains(out, "profile") || !strings.Contains(out, "insights") || !strings.Contains(out, "mentions") {
		t.Fatalf("schema missing expected nouns: %s", out)
	}
}

func TestDidYouMean(t *testing.T) {
	useTempStore(t)
	_, errb, code := run(t, "pst", "list")
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (usage)", code)
	}
	if !strings.Contains(errb, "did you mean") || !strings.Contains(errb, "post") {
		t.Fatalf("missing suggestion: %s", errb)
	}
}

// reply hide/unhide are idempotent: re-running reports changed=false, not an error (contract §9).
func TestIdempotentHideUnhide(t *testing.T) {
	useTempStore(t)
	if _, _, code := run(t, "reply", "hide", "42", "--allow-mutations", "--json"); code != 0 {
		t.Fatalf("hide exit = %d, want 0", code)
	}
	out, _, code := run(t, "reply", "hide", "42", "--allow-mutations", "--json")
	if code != 0 {
		t.Fatalf("re-hide exit = %d, want 0 (idempotent)", code)
	}
	if !strings.Contains(out, "\"changed\": false") {
		t.Fatalf("re-hide should report changed=false: %s", out)
	}
}

// Agent self-description must ship embedded in the binary.
func TestAgentPrintsSkill(t *testing.T) {
	useTempStore(t)
	out, _, code := run(t, "agent")
	if code != 0 {
		t.Fatalf("agent exit = %d, want 0", code)
	}
	if !strings.Contains(out, "knit") || !strings.Contains(out, "read-only by default") {
		t.Fatalf("agent output missing embedded SKILL.md: %s", out)
	}
}
