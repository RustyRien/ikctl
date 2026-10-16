package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTripAndPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.yaml")
	cfg := &Config{CurrentContext: "dev"}
	cfg.Upsert(Context{Name: "prod", Server: "https://prod", Token: "t2"})
	cfg.Upsert(Context{Name: "dev", Server: "http://localhost:7777", Token: "t1", User: "a@example.com"})
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("file mode = %v, want 0600", st.Mode().Perm())
	}
	if dst, _ := os.Stat(filepath.Dir(path)); dst.Mode().Perm() != 0o700 {
		t.Errorf("dir mode = %v, want 0700", dst.Mode().Perm())
	}
	if err := CheckPermissions(path); err != nil {
		t.Error(err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentContext != "dev" || len(got.Contexts) != 2 || got.Contexts[0].Name != "dev" {
		t.Fatalf("unexpected config %+v", got)
	}
	cur, ok := got.Current()
	if !ok || cur.Token != "t1" {
		t.Errorf("Current() = %+v, %v", cur, ok)
	}

	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckPermissions(path); err == nil {
		t.Error("expected a permission warning for 0644")
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil || len(cfg.Contexts) != 0 {
		t.Fatalf("Load(missing) = %+v, %v", cfg, err)
	}
}

func TestDelete(t *testing.T) {
	cfg := &Config{CurrentContext: "a", Contexts: []Context{{Name: "a"}, {Name: "b"}}}
	if !cfg.Delete("a") || cfg.CurrentContext != "" || len(cfg.Contexts) != 1 {
		t.Errorf("Delete(a) left %+v", cfg)
	}
	if cfg.Delete("zzz") {
		t.Error("Delete(zzz) should report false")
	}
}

func TestResolvePrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &Config{CurrentContext: "dev", Contexts: []Context{
		{Name: "dev", Server: "http://dev", Token: "dev-token"},
		{Name: "prod", Server: "https://prod", Token: "prod-token"},
	}}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	for _, env := range []string{EnvServer, EnvToken, EnvContext} {
		t.Setenv(env, "")
	}

	r, err := Resolve(Overrides{ConfigPath: path})
	if err != nil || r.Server != "http://dev" || r.Token != "dev-token" || r.ContextName != "dev" {
		t.Fatalf("file: %+v, %v", r, err)
	}

	t.Setenv(EnvContext, "prod")
	t.Setenv(EnvToken, "env-token")
	r, err = Resolve(Overrides{ConfigPath: path})
	if err != nil || r.Server != "https://prod" || r.Token != "env-token" {
		t.Fatalf("env: %+v, %v", r, err)
	}

	r, err = Resolve(Overrides{ConfigPath: path, Context: "dev", Server: "http://flag", Token: "flag-token"})
	if err != nil || r.Server != "http://flag" || r.Token != "flag-token" || r.ContextName != "dev" {
		t.Fatalf("flags: %+v, %v", r, err)
	}

	if _, err := Resolve(Overrides{ConfigPath: path, Context: "missing"}); err == nil {
		t.Error("unknown explicit context should fail")
	}
}

func TestDefaultPath(t *testing.T) {
	t.Setenv(EnvConfig, "")
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	if p, _ := DefaultPath(); p != "/xdg/ikctl/config.yaml" {
		t.Errorf("DefaultPath() = %s", p)
	}
	t.Setenv(EnvConfig, "/custom.yaml")
	if p, _ := DefaultPath(); p != "/custom.yaml" {
		t.Errorf("DefaultPath() = %s", p)
	}
}
