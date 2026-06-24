package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authorizeURL     = "https://threads.net/oauth/authorize"
	tokenExchangeURL = "https://graph.threads.net/oauth/access_token"
	longLivedURL     = "https://graph.threads.net/access_token"
	refreshURL       = "https://graph.threads.net/refresh_access_token"
)

// DefaultScopes is the read-leaning default; publishing/managing scopes are additive.
var DefaultScopes = []string{
	"threads_basic",
	"threads_content_publish",
	"threads_read_replies",
	"threads_manage_replies",
	"threads_manage_insights",
}

// OAuth holds the Threads app credentials and the configured redirect.
type OAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	HTTP         *http.Client
}

func (o *OAuth) httpClient() *http.Client {
	if o.HTTP != nil {
		return o.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// AuthorizeURL builds the consent URL the user opens in a browser (scopes comma-separated).
func (o *OAuth) AuthorizeURL(scopes []string, state string) string {
	q := url.Values{}
	q.Set("client_id", o.ClientID)
	q.Set("redirect_uri", o.RedirectURI)
	q.Set("scope", strings.Join(scopes, ","))
	q.Set("response_type", "code")
	if state != "" {
		q.Set("state", state)
	}
	return authorizeURL + "?" + q.Encode()
}

// ParseCallback extracts the authorization code from either a bare code or a full redirected
// URL (e.g. "https://localhost/cb?code=ABC#_"). Trailing "#_" and whitespace are stripped.
// If wantState is non-empty, the URL's state must match (CSRF check).
func ParseCallback(input, wantState string) (string, error) {
	s := strings.TrimSpace(input)
	s = strings.TrimSuffix(s, "#_")
	if !strings.Contains(s, "?") && !strings.Contains(s, "=") {
		return s, nil // bare code
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("could not parse callback: %w", err)
	}
	q := u.Query()
	if e := q.Get("error"); e != "" {
		return "", fmt.Errorf("authorization denied: %s %s", e, q.Get("error_description"))
	}
	code := q.Get("code")
	if code == "" {
		return "", fmt.Errorf("no ?code= found in callback")
	}
	if wantState != "" && q.Get("state") != wantState {
		return "", fmt.Errorf("state mismatch (possible CSRF); expected %q", wantState)
	}
	return strings.TrimSuffix(code, "#_"), nil
}

// tokenResp covers both the short-lived exchange and the long-lived/refresh responses.
type tokenResp struct {
	AccessToken string `json:"access_token"`
	UserID      any    `json:"user_id"` // number on short-lived exchange
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// ExchangeCode swaps an authorization code for a short-lived token (+ user id).
func (o *OAuth) ExchangeCode(ctx context.Context, code string) (*Credentials, error) {
	form := url.Values{}
	form.Set("client_id", o.ClientID)
	form.Set("client_secret", o.ClientSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", o.RedirectURI)
	form.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenExchangeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tr, err := o.do(req)
	if err != nil {
		return nil, err
	}
	return &Credentials{AccessToken: tr.AccessToken, UserID: userIDString(tr.UserID), TokenType: tr.TokenType}, nil
}

// ExchangeLongLived upgrades a short-lived token to a 60-day long-lived token.
func (o *OAuth) ExchangeLongLived(ctx context.Context, shortToken string) (*Credentials, error) {
	q := url.Values{}
	q.Set("grant_type", "th_exchange_token")
	q.Set("client_secret", o.ClientSecret)
	q.Set("access_token", shortToken)
	return o.getToken(ctx, longLivedURL+"?"+q.Encode())
}

// Refresh extends an unexpired long-lived token (must be ≥24h old, <60d) to a fresh 60 days.
func (o *OAuth) Refresh(ctx context.Context, longToken string) (*Credentials, error) {
	q := url.Values{}
	q.Set("grant_type", "th_refresh_token")
	q.Set("access_token", longToken)
	return o.getToken(ctx, refreshURL+"?"+q.Encode())
}

func (o *OAuth) getToken(ctx context.Context, u string) (*Credentials, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	tr, err := o.do(req)
	if err != nil {
		return nil, err
	}
	c := &Credentials{AccessToken: tr.AccessToken, TokenType: tr.TokenType}
	if tr.ExpiresIn > 0 {
		c.ExpiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
	return c, nil
}

func (o *OAuth) do(req *http.Request) (*tokenResp, error) {
	resp, err := o.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var tr tokenResp
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("could not parse token response: %w", err)
	}
	if tr.AccessToken == "" {
		return nil, fmt.Errorf("token endpoint returned no access_token")
	}
	return &tr, nil
}

func userIDString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%.0f", t)
	case json.Number:
		return t.String()
	case nil:
		return "me"
	default:
		return fmt.Sprintf("%v", t)
	}
}
