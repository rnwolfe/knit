// Package auth stores and loads Threads credentials and drives the OAuth flows.
// Tokens live in the OS keyring (99designs/keyring) with a 0600 XDG file fallback; the
// KNIT_TOKEN env var is honored first for ephemeral/headless use. Secrets never touch argv.
// See contract §7 and spec.md §Auth.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/99designs/keyring"
)

const (
	serviceName = "knit"
	credKey     = "credentials"
)

// Credentials is the persisted auth state for a single Threads account.
type Credentials struct {
	AccessToken string    `json:"accessToken"`
	UserID      string    `json:"userID"`
	TokenType   string    `json:"tokenType,omitempty"`
	ExpiresAt   time.Time `json:"expiresAt,omitempty"`
	Scopes      []string  `json:"scopes,omitempty"`
	// Source records where the creds came from ("env", "keyring", "file"); not persisted.
	Source string `json:"-"`
}

// Expired reports whether a known expiry is in the past. Zero ExpiresAt = unknown → not expired.
func (c *Credentials) Expired() bool {
	return !c.ExpiresAt.IsZero() && time.Now().After(c.ExpiresAt)
}

// DaysUntilExpiry returns whole days until expiry, or -1 if unknown.
func (c *Credentials) DaysUntilExpiry() int {
	if c.ExpiresAt.IsZero() {
		return -1
	}
	return int(time.Until(c.ExpiresAt).Hours() / 24)
}

// Load resolves credentials with precedence: KNIT_TOKEN env → OS keyring → 0600 file.
// Returns (nil, nil) when nothing is configured (callers raise AUTH_REQUIRED).
func Load() (*Credentials, error) {
	if tok := os.Getenv("KNIT_TOKEN"); tok != "" {
		uid := os.Getenv("KNIT_USER_ID")
		if uid == "" {
			uid = "me"
		}
		return &Credentials{AccessToken: tok, UserID: uid, Source: "env"}, nil
	}
	if kr, err := openKeyring(); err == nil {
		if item, err := kr.Get(credKey); err == nil {
			var c Credentials
			if err := json.Unmarshal(item.Data, &c); err == nil {
				c.Source = "keyring"
				return &c, nil
			}
		}
	}
	return loadFile()
}

// Save persists credentials to the keyring, falling back to a 0600 file.
func Save(c *Credentials) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if kr, err := openKeyring(); err == nil {
		if err := kr.Set(keyring.Item{Key: credKey, Data: data, Label: "knit Threads credentials"}); err == nil {
			return nil
		}
	}
	return saveFile(data)
}

// Clear removes locally stored credentials from both the keyring and the file fallback.
// It does NOT revoke the token upstream (Threads has no token-revocation endpoint).
func Clear() error {
	var firstErr error
	if kr, err := openKeyring(); err == nil {
		if err := kr.Remove(credKey); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
			firstErr = err
		}
	}
	if err := os.Remove(filePath()); err != nil && !os.IsNotExist(err) && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func openKeyring() (keyring.Keyring, error) {
	// Only OS-native backends; we provide our own 0600 file fallback (keyring's FileBackend
	// would prompt for a passphrase, which deadlocks an agent).
	return keyring.Open(keyring.Config{
		ServiceName:              serviceName,
		KeychainTrustApplication: true,
		AllowedBackends: []keyring.BackendType{
			keyring.KeychainBackend,
			keyring.SecretServiceBackend,
			keyring.WinCredBackend,
		},
	})
}

// --- 0600 file fallback -----------------------------------------------------

func filePath() string {
	if p := os.Getenv("KNIT_CREDENTIALS_FILE"); p != "" {
		return p
	}
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "knit", "credentials.json")
}

func loadFile() (*Credentials, error) {
	path := filePath()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if fi, err := os.Stat(path); err == nil && fi.Mode().Perm()&0o077 != 0 {
		fmt.Fprintf(os.Stderr, "warning: %s has insecure permissions %o (want 0600)\n", path, fi.Mode().Perm())
	}
	var c Credentials
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	c.Source = "file"
	return &c, nil
}

func saveFile(data []byte) error {
	path := filePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
