package tokenfile

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "token.json")

	if _, err := Load(path); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	want := &oauth2.Token{AccessToken: "a", RefreshToken: "r", Expiry: time.Unix(1000, 0).UTC()}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "a" || got.RefreshToken != "r" || !got.Expiry.Equal(want.Expiry) {
		t.Fatalf("got %+v", got)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(path)
		if info.Mode().Perm() != filePerm {
			t.Fatalf("perm %o", info.Mode().Perm())
		}
	}
}
