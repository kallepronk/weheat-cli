package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kallepronk/weheat-cli/internal/tokenfile"
	"golang.org/x/oauth2"
)

func TestParseRange(t *testing.T) {
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)

	s, e, err := parseRange("", "", now)
	if err != nil || !e.Equal(now) || !s.Equal(now.Add(-defaultWindow)) {
		t.Fatalf("defaults: %v %v %v", s, e, err)
	}
	s, e, err = parseRange("2026-09-01T00:00:00Z", "2026-09-02T00:00:00Z", now)
	if err != nil || s.Day() != 1 || e.Day() != 2 {
		t.Fatalf("explicit: %v %v %v", s, e, err)
	}
	if _, _, err = parseRange("2026-09-03T00:00:00Z", "2026-09-02T00:00:00Z", now); err == nil {
		t.Fatal("inverted range should fail")
	}
	if _, _, err = parseRange("yesterday", "", now); err == nil {
		t.Fatal("garbage should fail")
	}
}

// TestRunEndToEnd runs `weheat me` against a fake API using a stored, still-valid token.
func TestRunEndToEnd(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/me" || r.Header.Get("Authorization") != "Bearer access" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"id":"u1","role":1,"updatedOn":"2026-09-04T00:00:00Z","createdOn":"2026-09-04T00:00:00Z"}`))
	}))
	defer api.Close()

	tokenPath := filepath.Join(t.TempDir(), "token.json")
	err := tokenfile.Save(tokenPath, &oauth2.Token{
		AccessToken: "access", RefreshToken: "refresh", Expiry: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("WEHEAT_CLIENT_ID", "")
	t.Setenv("WEHEAT_CLIENT_SECRET", "")
	t.Setenv("WEHEAT_BASE_URL", api.URL)
	t.Setenv("WEHEAT_REFRESH_TOKEN", "")

	var stdout, stderr bytes.Buffer
	if err := run([]string{"--token-file", tokenPath, "--no-rate-limit", "me"}, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v (stderr %q)", err, stderr.String())
	}
	var out map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil || out["id"] != "u1" {
		t.Fatalf("stdout %q err %v", stdout.String(), err)
	}
}

func TestRunUsageOnNoCommand(t *testing.T) {
	var stderr bytes.Buffer
	if err := run(nil, &bytes.Buffer{}, &stderr); err == nil || !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("err %v stderr %q", err, stderr.String())
	}
}
