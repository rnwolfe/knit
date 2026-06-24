package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rnwolfe/knit/internal/errs"
)

func newTestClient(t *testing.T, h http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New("test-token", "me").WithBaseURL(srv.URL), srv
}

func exitCode(t *testing.T, err error) (int, string) {
	t.Helper()
	var ce *errs.CLIError
	if !errors.As(err, &ce) {
		t.Fatalf("error is not *CLIError: %v", err)
	}
	return ce.Exit, ce.Code
}

func TestListPostsMapsAndPaginates(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("access_token") != "test-token" {
			t.Errorf("missing access_token query param")
		}
		if r.URL.Query().Get("fields") == "" {
			t.Errorf("missing fields param")
		}
		w.Write([]byte(`{
			"data":[{"id":"1","username":"me","text":"hi","media_type":"TEXT_POST","is_quote_post":false,"permalink":"https://t/1"}],
			"paging":{"cursors":{"after":"CUR"},"next":"https://graph.threads.net/next"}
		}`))
	})
	posts, cursor, err := c.ListPosts(context.Background(), PageOpts{Limit: 10})
	if err != nil {
		t.Fatalf("ListPosts: %v", err)
	}
	if len(posts) != 1 || posts[0].MediaType != "TEXT_POST" || posts[0].Text == nil || *posts[0].Text != "hi" {
		t.Fatalf("bad mapping: %+v", posts)
	}
	if cursor != "CUR" {
		t.Fatalf("cursor = %q, want CUR", cursor)
	}
}

func TestNoNextCursorAtEnd(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[],"paging":{"cursors":{"after":"X"}}}`)) // no paging.next
	})
	_, cursor, err := c.ListPosts(context.Background(), PageOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if cursor != "" {
		t.Fatalf("cursor = %q, want empty at end-of-results", cursor)
	}
}

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		body     string
		wantExit int
		wantCode string
	}{
		{"rate429", http.StatusTooManyRequests, `{"error":{"message":"limit","code":4}}`, errs.ExitRate, "RATE_LIMITED"},
		{"authInvalidToken", http.StatusBadRequest, `{"error":{"message":"bad token","code":190}}`, errs.ExitAuth, "AUTH_REQUIRED"},
		{"permission", http.StatusBadRequest, `{"error":{"message":"no perm","code":200}}`, errs.ExitPerm, "PERMISSION_DENIED"},
		{"notFound", http.StatusNotFound, `{"error":{"message":"gone","code":803}}`, errs.ExitNotFound, "NOT_FOUND"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
			})
			_, err := c.GetPost(context.Background(), "1")
			exit, code := exitCode(t, err)
			if exit != tc.wantExit || code != tc.wantCode {
				t.Fatalf("got exit=%d code=%s, want exit=%d code=%s", exit, code, tc.wantExit, tc.wantCode)
			}
		})
	}
}

func TestAuthRequiredWhenNoToken(t *testing.T) {
	c := New("", "me")
	_, err := c.GetPost(context.Background(), "1")
	exit, code := exitCode(t, err)
	if exit != errs.ExitAuth || code != "AUTH_REQUIRED" {
		t.Fatalf("got exit=%d code=%s, want AUTH_REQUIRED/4", exit, code)
	}
}

// DeletePost is idempotent: a 404 target is a soft success (existed=false), per contract §9.
func TestDeleteIdempotentOnNotFound(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"message":"gone","code":803}}`))
	})
	res, err := c.DeletePost(context.Background(), "1")
	if err != nil {
		t.Fatalf("delete should be idempotent on 404, got: %v", err)
	}
	if !res.OK || res.Existed {
		t.Fatalf("want ok=true existed=false, got %+v", res)
	}
}

func TestSearchScopeDetection(t *testing.T) {
	selfUsernameCache = "" // reset memo
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "keyword_search") {
			// a post by someone else → scope should be "public".
			w.Write([]byte(`{"data":[{"id":"9","username":"someone_else","text":"x","media_type":"TEXT_POST"}]}`))
			return
		}
		// the /me profile lookup scope-detection performs.
		w.Write([]byte(`{"id":"meid","username":"me_user"}`))
	})
	_, _, scope, err := c.Search(context.Background(), SearchOpts{Query: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if scope != "public" {
		t.Fatalf("scope = %q, want public", scope)
	}
}
