package cli

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"strings"

	"github.com/rnwolfe/knit/internal/api"
	"github.com/rnwolfe/knit/internal/auth"
	"github.com/rnwolfe/knit/internal/errs"
)

// AuthCmd wires the Threads OAuth model (spec.md §Auth): --token-stdin (headless) +
// paste-the-callback-URL (human onboarding), long-lived 60d token with refresh. Secrets come
// via stdin/env (never argv) and persist in the OS keyring + 0600 fallback (contract §7).
type AuthCmd struct {
	Status  AuthStatusCmd  `cmd:"" help:"Test the token; report account, expiry, and source."`
	Login   AuthLoginCmd   `cmd:"" help:"Authenticate (paste-the-callback-URL, or --token-stdin)."`
	Logout  AuthLogoutCmd  `cmd:"" help:"Remove stored credentials from the keyring/file (local only)."`
	Refresh AuthRefreshCmd `cmd:"" help:"Extend the long-lived (60-day) token; safe on a schedule."`
}

// oauthFromEnv builds the OAuth config from KNIT_CLIENT_ID / KNIT_CLIENT_SECRET /
// KNIT_REDIRECT_URI (never argv). client_id/secret are the Threads app's, not the Meta app's.
func oauthFromEnv() (*auth.OAuth, error) {
	id, secret := os.Getenv("KNIT_CLIENT_ID"), os.Getenv("KNIT_CLIENT_SECRET")
	redirect := os.Getenv("KNIT_REDIRECT_URI")
	if redirect == "" {
		redirect = "https://localhost/knit/callback"
	}
	if id == "" || secret == "" {
		return nil, errs.New(errs.ExitConfig, "CONFIG_ERROR",
			"KNIT_CLIENT_ID and KNIT_CLIENT_SECRET must be set for browser login",
			"export the Threads app credentials, or use `knit auth login --token-stdin`")
	}
	return &auth.OAuth{ClientID: id, ClientSecret: secret, RedirectURI: redirect}, nil
}

// --- auth login -------------------------------------------------------------

type AuthLoginCmd struct {
	TokenStdin bool `name:"token-stdin" help:"Read a pre-obtained token from stdin (headless; no browser)."`
}

func (c *AuthLoginCmd) Run(rt *Runtime) error {
	if c.TokenStdin {
		return c.loginToken(rt)
	}
	return c.loginBrowser(rt)
}

// loginToken stores a token piped on stdin, upgrading to long-lived when a client secret is set.
func (c *AuthLoginCmd) loginToken(rt *Runtime) error {
	raw, err := io.ReadAll(rt.Stdin)
	if err != nil {
		return errs.New(errs.ExitGeneric, "READ_ERROR", err.Error(), "")
	}
	token := strings.TrimSpace(string(raw))
	if token == "" {
		return errs.InputRequired("token on stdin")
	}
	creds := &auth.Credentials{AccessToken: token, UserID: "me"}
	// Best-effort upgrade to a 60-day token if app credentials are present.
	if o, oerr := oauthFromEnv(); oerr == nil {
		if long, lerr := o.ExchangeLongLived(rt.Ctx, token); lerr == nil {
			creds.AccessToken, creds.ExpiresAt, creds.TokenType = long.AccessToken, long.ExpiresAt, long.TokenType
		}
	}
	return validateAndSave(rt, creds, "token-stdin")
}

// loginBrowser prints the authorize URL, reads the pasted callback URL/code from stdin, and
// exchanges it. No server, no cert (Meta requires an HTTPS callback; we only need the code).
func (c *AuthLoginCmd) loginBrowser(rt *Runtime) error {
	if rt.Cfg.NoInput {
		return errs.New(errs.ExitInputRequired, "INPUT_REQUIRED",
			"browser login needs a pasted callback URL", "use `knit auth login --token-stdin` for headless auth")
	}
	o, err := oauthFromEnv()
	if err != nil {
		return err
	}
	state := randomState()
	rt.Out.Info("1. Open this URL, approve, then paste the redirected URL (or the code) here:\n\n   %s\n",
		o.AuthorizeURL(auth.DefaultScopes, state))
	rt.Out.Info("2. Paste callback URL/code, then press Enter:")

	raw, err := io.ReadAll(rt.Stdin)
	if err != nil {
		return errs.New(errs.ExitGeneric, "READ_ERROR", err.Error(), "")
	}
	code, err := auth.ParseCallback(string(raw), state)
	if err != nil {
		return errs.New(errs.ExitUsage, "CALLBACK_PARSE", err.Error(), "paste the full redirected URL including ?code=")
	}
	short, err := o.ExchangeCode(rt.Ctx, code)
	if err != nil {
		return errs.New(errs.ExitAuth, "EXCHANGE_FAILED", err.Error(), "the code may be expired (1h TTL); restart `knit auth login`")
	}
	creds := short
	if long, lerr := o.ExchangeLongLived(rt.Ctx, short.AccessToken); lerr == nil {
		creds = &auth.Credentials{AccessToken: long.AccessToken, UserID: short.UserID, TokenType: long.TokenType, ExpiresAt: long.ExpiresAt}
	}
	return validateAndSave(rt, creds, "browser")
}

// validateAndSave confirms the token works (resolves the user id) and persists it.
func validateAndSave(rt *Runtime, creds *auth.Credentials, method string) error {
	client := api.New(creds.AccessToken, creds.UserID)
	profile, err := client.Profile(rt.Ctx, "me")
	if err != nil {
		return err
	}
	creds.UserID = profile.ID
	if err := auth.Save(creds); err != nil {
		return errs.New(errs.ExitConfig, "SAVE_ERROR", err.Error(), "check keyring availability or file perms")
	}
	out := map[string]any{"ok": true, "method": method, "userId": profile.ID, "username": profile.Username}
	if d := creds.DaysUntilExpiry(); d >= 0 {
		out["expiresInDays"] = d
	}
	return rt.Out.Emit(out)
}

// --- auth status ------------------------------------------------------------

type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(rt *Runtime) error {
	if rt.Creds == nil || rt.Creds.AccessToken == "" {
		return errs.New(errs.ExitAuth, "AUTH_REQUIRED", "not authenticated",
			"run `knit auth login` (or set KNIT_TOKEN)")
	}
	// Actively test the token.
	profile, err := rt.API.Profile(rt.Ctx, "me")
	if err != nil {
		return err
	}
	out := map[string]any{
		"authenticated": true,
		"userId":        profile.ID,
		"username":      profile.Username,
		"source":        rt.Creds.Source,
		"token":         redact(rt.Creds.AccessToken),
	}
	if d := rt.Creds.DaysUntilExpiry(); d >= 0 {
		out["expiresInDays"] = d
	}
	if q, err := rt.API.PublishingLimit(rt.Ctx); err == nil {
		out["publishQuotaRemaining"] = q.Remaining
	}
	return rt.Out.Emit(out)
}

// --- auth logout ------------------------------------------------------------

type AuthLogoutCmd struct{}

func (c *AuthLogoutCmd) Run(rt *Runtime) error {
	if err := auth.Clear(); err != nil {
		return errs.New(errs.ExitConfig, "LOGOUT_ERROR", err.Error(), "")
	}
	rt.Out.Info("removed local credentials (the token is not revoked upstream — Threads has no revoke endpoint)")
	return rt.Out.Emit(map[string]any{"ok": true, "scope": "local"})
}

// --- auth refresh -----------------------------------------------------------

type AuthRefreshCmd struct{}

func (c *AuthRefreshCmd) Run(rt *Runtime) error {
	if rt.Creds == nil || rt.Creds.AccessToken == "" {
		return errs.New(errs.ExitAuth, "AUTH_REQUIRED", "not authenticated", "run `knit auth login`")
	}
	// Refresh needs only the long-lived token (grant_type=th_refresh_token), no client secret.
	o := &auth.OAuth{}
	long, err := o.Refresh(rt.Ctx, rt.Creds.AccessToken)
	if err != nil {
		return errs.New(errs.ExitAuth, "REFRESH_FAILED", err.Error(),
			"token must be ≥24h old and unexpired; otherwise re-run `knit auth login`")
	}
	rt.Creds.AccessToken = long.AccessToken
	rt.Creds.ExpiresAt = long.ExpiresAt
	if err := auth.Save(rt.Creds); err != nil {
		return errs.New(errs.ExitConfig, "SAVE_ERROR", err.Error(), "")
	}
	out := map[string]any{"ok": true}
	if d := rt.Creds.DaysUntilExpiry(); d >= 0 {
		out["expiresInDays"] = d
	}
	return rt.Out.Emit(out)
}

// --- helpers ----------------------------------------------------------------

func redact(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "…" + token[len(token)-4:]
}

func randomState() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "knitstate"
	}
	return hex.EncodeToString(b)
}
