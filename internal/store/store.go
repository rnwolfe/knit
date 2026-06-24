// Package store is a PLACEHOLDER target: a local JSON-backed stand-in for the Threads
// API so the scaffold compiles, runs, and is fully testable offline. cli-implement
// REPLACES this package with a real Threads API client (internal/api) and wires auth
// (internal/auth with the OS keyring, per contract §7). Nothing here talks to Threads.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// Post is a trimmed stand-in for a Threads post/reply. The real client (cli-implement)
// returns the full schema from spec.md; these few fields are enough to exercise the
// contract surface offline.
type Post struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Text        string `json:"text"`
	MediaType   string `json:"mediaType"`
	Permalink   string `json:"permalink"`
	Timestamp   string `json:"timestamp"`
	IsQuotePost bool   `json:"isQuotePost"`
}

// state is the on-disk shape: authored posts plus the hidden-reply set (idempotent flag).
type state struct {
	Posts      []Post          `json:"posts"`
	HiddenReps map[string]bool `json:"hiddenReplies"`
}

type Store struct{ path string }

func New(path string) *Store { return &Store{path: path} }

// DefaultPath resolves the placeholder store location (XDG-aware), overridable via
// KNIT_STORE. The real client keeps tokens in the keyring, not a JSON file.
func DefaultPath() string {
	if p := os.Getenv("KNIT_STORE"); p != "" {
		return p
	}
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "knit", "store.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "knit", "store.json")
}

func (s *Store) load() (state, error) {
	st := state{Posts: []Post{}, HiddenReps: map[string]bool{}}
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return st, err
	}
	if st.Posts == nil {
		st.Posts = []Post{}
	}
	if st.HiddenReps == nil {
		st.HiddenReps = map[string]bool{}
	}
	return st, nil
}

func (s *Store) save(st state) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}

// ListPosts returns the authored posts (placeholder for `GET /{user}/threads`).
func (s *Store) ListPosts() ([]Post, error) {
	st, err := s.load()
	return st.Posts, err
}

// GetPost returns one post by id (placeholder for `GET /{media-id}`).
func (s *Store) GetPost(id string) (Post, bool, error) {
	st, err := s.load()
	if err != nil {
		return Post{}, false, err
	}
	for _, p := range st.Posts {
		if p.ID == id {
			return p, true, nil
		}
	}
	return Post{}, false, nil
}

// CreatePost appends a post with a deterministic next-integer id (placeholder for the
// two-step publish flow). The real client returns the published media id + permalink.
func (s *Store) CreatePost(text string) (Post, error) {
	st, err := s.load()
	if err != nil {
		return Post{}, err
	}
	max := 0
	for _, p := range st.Posts {
		if n, err := strconv.Atoi(p.ID); err == nil && n > max {
			max = n
		}
	}
	p := Post{
		ID:        strconv.Itoa(max + 1),
		Username:  "me",
		Text:      text,
		MediaType: "TEXT_POST",
		Permalink: "https://www.threads.net/@me/post/" + strconv.Itoa(max+1),
		Timestamp: "1970-01-01T00:00:00Z",
	}
	st.Posts = append(st.Posts, p)
	return p, s.save(st)
}

// SetReplyHidden sets a reply's hidden state. It is idempotent (contract §9): setting the
// value it already has is a soft success. Reports whether the state changed.
func (s *Store) SetReplyHidden(replyID string, hidden bool) (changed bool, err error) {
	st, err := s.load()
	if err != nil {
		return false, err
	}
	if st.HiddenReps[replyID] == hidden {
		return false, nil
	}
	st.HiddenReps[replyID] = hidden
	return true, s.save(st)
}
