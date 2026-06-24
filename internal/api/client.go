// Package api is the real Threads API client (direct HTTP to graph.threads.net). It maps
// Graph errors onto the tool's stable exit codes, honors Retry-After / rate-limit signals,
// and retries transient failures with backoff. Tokens are sent as the access_token query
// param (per Threads docs). See spec.md §Verified API facts.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rnwolfe/knit/internal/errs"
)

const defaultBaseURL = "https://graph.threads.net/v1.0"

// Client talks to the Threads Graph API for one authenticated user.
type Client struct {
	token   string
	userID  string // numeric id or "me"
	baseURL string
	http    *http.Client
}

// New constructs a client. userID may be "me". An empty token is allowed (calls will fail
// with AUTH_REQUIRED) so doctor/status can construct a client without credentials.
func New(token, userID string) *Client {
	if userID == "" {
		userID = "me"
	}
	return &Client{
		token:   token,
		userID:  userID,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// WithBaseURL overrides the API base (for tests).
func (c *Client) WithBaseURL(u string) *Client { c.baseURL = u; return c }

// WithHTTP overrides the HTTP client (for tests).
func (c *Client) WithHTTP(h *http.Client) *Client { c.http = h; return c }

func (c *Client) requireToken() error {
	if c.token == "" {
		return errs.New(errs.ExitAuth, "AUTH_REQUIRED", "no Threads credentials found",
			"run `knit auth login` (or set KNIT_TOKEN)")
	}
	return nil
}

// graphError is the standard Graph API error envelope.
type graphError struct {
	Error struct {
		Message   string `json:"message"`
		Type      string `json:"type"`
		Code      int    `json:"code"`
		Subcode   int    `json:"error_subcode"`
		FBTraceID string `json:"fbtrace_id"`
	} `json:"error"`
}

// get performs a GET against /{path} with query params, decoding the body into out.
func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	return c.do(ctx, http.MethodGet, path, q, nil, out)
}

// postForm performs a POST with form-encoded params (Graph writes take query/form params).
func (c *Client) postForm(ctx context.Context, path string, q url.Values, out any) error {
	return c.do(ctx, http.MethodPost, path, q, nil, out)
}

// del performs a DELETE.
func (c *Client) del(ctx context.Context, path string, q url.Values, out any) error {
	return c.do(ctx, http.MethodDelete, path, q, nil, out)
}

func (c *Client) do(ctx context.Context, method, path string, q url.Values, body io.Reader, out any) error {
	if err := c.requireToken(); err != nil {
		return err
	}
	if q == nil {
		q = url.Values{}
	}
	q.Set("access_token", c.token)
	full := c.baseURL + "/" + strings.TrimPrefix(path, "/") + "?" + q.Encode()

	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, full, body)
		if err != nil {
			return errs.New(errs.ExitGeneric, "REQUEST_ERROR", err.Error(), "")
		}
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = errs.New(errs.ExitRetry, "NETWORK", err.Error(), "check connectivity and retry")
			c.backoff(ctx, attempt)
			continue
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode/100 == 2 {
			if out == nil || len(data) == 0 {
				return nil
			}
			if err := json.Unmarshal(data, out); err != nil {
				return errs.New(errs.ExitGeneric, "PARSE_ERROR", "could not parse API response: "+err.Error(), "")
			}
			return nil
		}

		ce := mapError(resp, data)
		// Retry only transient 5xx (RETRYABLE); auth/rate/perm/not-found are terminal.
		if ce.Exit == errs.ExitRetry && attempt < maxAttempts {
			lastErr = ce
			c.backoff(ctx, attempt)
			continue
		}
		return ce
	}
	return lastErr
}

func (c *Client) backoff(ctx context.Context, attempt int) {
	d := time.Duration(attempt*attempt) * 200 * time.Millisecond
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

// mapError converts an HTTP error response into a structured CLIError with a stable exit code.
func mapError(resp *http.Response, body []byte) *errs.CLIError {
	var ge graphError
	_ = json.Unmarshal(body, &ge)
	msg := ge.Error.Message
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	switch {
	case resp.StatusCode == http.StatusTooManyRequests || isRateCode(ge.Error.Code):
		rem := retryAfter(resp)
		fix := "back off and retry later (Threads caps publishing at 250/24h, search at 2200/24h)"
		if rem != "" {
			fix = "retry after " + rem
		}
		return errs.New(errs.ExitRate, "RATE_LIMITED", msg, fix)
	case ge.Error.Code == 190 || resp.StatusCode == http.StatusUnauthorized:
		return errs.New(errs.ExitAuth, "AUTH_REQUIRED", msg, "token invalid/expired — run `knit auth refresh` or `knit auth login`")
	case isPermCode(ge.Error.Code) || resp.StatusCode == http.StatusForbidden:
		return errs.New(errs.ExitPerm, "PERMISSION_DENIED", msg,
			"a required scope or advanced access is missing — check `knit auth status` and App Review")
	case resp.StatusCode == http.StatusNotFound:
		return errs.New(errs.ExitNotFound, "NOT_FOUND", msg, "verify the id exists and is visible to you")
	case resp.StatusCode/100 == 5:
		return errs.New(errs.ExitRetry, "RETRYABLE", msg, "transient upstream error — retry")
	default:
		return errs.New(errs.ExitGeneric, "API_ERROR", msg, "")
	}
}

func isRateCode(code int) bool {
	switch code {
	case 4, 17, 32, 613:
		return true
	}
	return false
}

func isPermCode(code int) bool {
	switch code {
	case 10, 200, 803:
		return true
	}
	return false
}

// retryAfter reads Retry-After or the X-Business-Use-Case-Usage regain estimate (minutes).
func retryAfter(resp *http.Response) string {
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil {
			return (time.Duration(secs) * time.Second).String()
		}
		return ra
	}
	if u := resp.Header.Get("X-Business-Use-Case-Usage"); u != "" {
		if strings.Contains(u, "estimated_time_to_regain_access") {
			return "the time noted in X-Business-Use-Case-Usage"
		}
	}
	return ""
}
