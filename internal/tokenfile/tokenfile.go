// Package tokenfile persists the OAuth token between CLI runs.
//
// Keycloak rotates refresh tokens on every refresh, so a CLI that does not write the new
// one back locks itself out after the first run.
package tokenfile

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

// ErrNotFound means no token has been stored yet.
var ErrNotFound = errors.New("no stored token; run `weheat login` or set WEHEAT_REFRESH_TOKEN")

const (
	dirPerm  = 0o700
	filePerm = 0o600
)

// DefaultPath is $XDG_CONFIG_HOME/weheat/token.json, falling back to ~/.config.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "weheat", "token.json"), nil
}

// Load reads the token at path.
func Load(path string) (*oauth2.Token, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var tok oauth2.Token
	if err := json.Unmarshal(b, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

// Save writes the token to path with owner-only permissions, creating parent directories.
func Save(path string, tok *oauth2.Token) error {
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return err
	}
	b, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, filePerm)
}
