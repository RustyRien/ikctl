package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.yaml")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	expiry := time.Date(2026, 7, 2, 10, 0, 0, 0, time.UTC)
	err = store.Put("https://ik.example", Credentials{
		Provider:     "github",
		RefreshToken: "refresh-1",
		Token:        "jwt-1",
		TokenExpiry:  expiry,
	})
	if err != nil {
		t.Fatalf("put credentials: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat credentials: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("credentials perm = %#o", info.Mode().Perm())
	}

	reloaded, err := OpenStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	cred, ok := reloaded.Get("https://ik.example")
	if !ok {
		t.Fatalf("missing credentials after reload")
	}
	if cred.Provider != "github" || cred.RefreshToken != "refresh-1" || cred.Token != "jwt-1" {
		t.Fatalf("credentials = %#v", cred)
	}
	if !cred.TokenExpiry.Equal(expiry) {
		t.Fatalf("token expiry = %v", cred.TokenExpiry)
	}
}
