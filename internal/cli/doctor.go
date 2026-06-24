package cli

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rnwolfe/knit/internal/errs"
	"github.com/rnwolfe/knit/internal/output"
)

// DoctorCmd runs real environment/auth/connectivity diagnostics.
type DoctorCmd struct {
	ForAgent bool `name:"for-agent" help:"Terser, machine-skimmable diagnostics for an agent."`
}

type check struct {
	Name   string `json:"name"`
	Status string `json:"status"` // ok | warn | fail
	Detail string `json:"detail"`
	Fix    string `json:"fix,omitempty"`
}

func (c *DoctorCmd) Run(rt *Runtime) error {
	checks := []check{
		appConfigCheck(),
		credentialsCheck(rt),
		connectivityCheck(),
		tokenCheck(rt),
	}

	failed := []string{}
	for _, ch := range checks {
		if ch.Status == "fail" {
			failed = append(failed, ch.Name)
		}
	}

	if c.ForAgent {
		out := map[string]any{"ok": len(failed) == 0}
		if len(failed) > 0 {
			out["failed"] = failed
		}
		if err := rt.Out.Emit(out); err != nil {
			return err
		}
	} else if rt.Out.Format == output.FormatPlain {
		for _, ch := range checks {
			fmt.Fprintf(rt.Out.Stdout, "%s %-13s %s\n", glyph(ch.Status), ch.Name, ch.Detail)
			if ch.Fix != "" && ch.Status != "ok" {
				fmt.Fprintf(rt.Out.Stdout, "    fix: %s\n", ch.Fix)
			}
		}
	} else {
		if err := rt.Out.Emit(checks); err != nil {
			return err
		}
	}

	if len(failed) > 0 {
		return errs.New(errs.ExitConfig, "DOCTOR_FAILED", "checks failed: "+strings.Join(failed, ","),
			"address each failing check's fix")
	}
	return nil
}

func appConfigCheck() check {
	id, secret := os.Getenv("KNIT_CLIENT_ID"), os.Getenv("KNIT_CLIENT_SECRET")
	if id != "" && secret != "" {
		return check{"config", "ok", "KNIT_CLIENT_ID and KNIT_CLIENT_SECRET set", ""}
	}
	// Not fatal: a stored/KNIT_TOKEN token works for reads/writes; app creds are only needed
	// for browser login and long-lived exchange.
	return check{"config", "warn", "KNIT_CLIENT_ID/SECRET not set",
		"set them for `knit auth login` / long-lived token exchange (not needed if using KNIT_TOKEN)"}
}

func credentialsCheck(rt *Runtime) check {
	if rt.Creds == nil || rt.Creds.AccessToken == "" {
		return check{"credentials", "fail", "no token found",
			"run `knit auth login` (or set KNIT_TOKEN)"}
	}
	detail := fmt.Sprintf("token present (source: %s)", rt.Creds.Source)
	if d := rt.Creds.DaysUntilExpiry(); d >= 0 {
		detail += fmt.Sprintf(", expires in %d days", d)
		if d < 7 {
			return check{"credentials", "warn", detail, "run `knit auth refresh` soon"}
		}
	}
	if rt.Creds.Expired() {
		return check{"credentials", "fail", "token expired", "run `knit auth login`"}
	}
	return check{"credentials", "ok", detail, ""}
}

func connectivityCheck() check {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get("https://graph.threads.net/v1.0/")
	if err != nil {
		return check{"connectivity", "fail", "cannot reach graph.threads.net: " + err.Error(),
			"check network/DNS/proxy"}
	}
	resp.Body.Close()
	return check{"connectivity", "ok", "graph.threads.net reachable", ""}
}

func tokenCheck(rt *Runtime) check {
	if rt.Creds == nil || rt.Creds.AccessToken == "" {
		return check{"auth", "fail", "no token to validate", "run `knit auth login`"}
	}
	if _, err := rt.API.Profile(rt.Ctx, "me"); err != nil {
		return check{"auth", "fail", "token rejected by Threads: " + err.Error(),
			"run `knit auth refresh` or `knit auth login`"}
	}
	return check{"auth", "ok", "token valid", ""}
}

func glyph(status string) string {
	switch status {
	case "ok":
		return "[✓]" // ✓
	case "warn":
		return "[!]"
	default:
		return "[✗]" // ✗
	}
}
