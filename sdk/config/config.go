// Package config manages the ikctl configuration file holding named contexts
// (server URL + token). The file lives at $IKCTL_CONFIG, or
// $XDG_CONFIG_HOME/ikctl/config.yaml, or ~/.config/ikctl/config.yaml, and is
// written with 0600 permissions because it contains credentials.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"sigs.k8s.io/yaml"
)

// Environment variables that override the config file.
const (
	EnvConfig  = "IKCTL_CONFIG"
	EnvServer  = "IKCTL_SERVER"
	EnvToken   = "IKCTL_TOKEN" //nolint:gosec // environment variable name, not a credential
	EnvContext = "IKCTL_CONTEXT"
)

const (
	dirName  = "ikctl"
	fileName = "config.yaml"
	fileMode = 0o600
	dirMode  = 0o700
)

// Context is a named server + credential pair.
type Context struct {
	Name                  string `json:"name"`
	Server                string `json:"server"`
	Token                 string `json:"token,omitempty"`
	User                  string `json:"user,omitempty"`
	InsecureSkipTLSVerify bool   `json:"insecure-skip-tls-verify,omitempty"`
}

// Config is the content of the config file.
type Config struct {
	CurrentContext string    `json:"current-context,omitempty"`
	Contexts       []Context `json:"contexts"`
}

// DefaultPath returns the config file path.
func DefaultPath() (string, error) {
	if p := os.Getenv(EnvConfig); p != "" {
		return p, nil
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locate home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, dirName, fileName), nil
}

// Load reads the config file. A missing file yields an empty config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is chosen by the user
	if errors.Is(err, fs.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.UnmarshalStrict(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return &cfg, nil
}

// Save writes the config file atomically with 0600 permissions.
func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".config-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op after a successful rename

	if err := tmp.Chmod(fileMode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod config: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

// CheckPermissions returns an error if the config file is readable by group or others.
func CheckPermissions(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if st.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("config file %s has permissions %v; it contains tokens and should be 0600 (chmod 600 %s)",
			path, st.Mode().Perm(), path)
	}
	return nil
}

// Get returns the context with the given name.
func (c *Config) Get(name string) (*Context, bool) {
	for i := range c.Contexts {
		if c.Contexts[i].Name == name {
			return &c.Contexts[i], true
		}
	}
	return nil, false
}

// Current returns the current context, if any.
func (c *Config) Current() (*Context, bool) {
	if c.CurrentContext == "" {
		return nil, false
	}
	return c.Get(c.CurrentContext)
}

// Upsert adds or replaces a context.
func (c *Config) Upsert(ctx Context) {
	if existing, ok := c.Get(ctx.Name); ok {
		*existing = ctx
		return
	}
	c.Contexts = append(c.Contexts, ctx)
	slices.SortFunc(c.Contexts, func(a, b Context) int { return strings.Compare(a.Name, b.Name) })
}

// Delete removes a context. It returns false if no such context exists.
func (c *Config) Delete(name string) bool {
	n := len(c.Contexts)
	c.Contexts = slices.DeleteFunc(c.Contexts, func(x Context) bool { return x.Name == name })
	if c.CurrentContext == name {
		c.CurrentContext = ""
	}
	return len(c.Contexts) != n
}

// Overrides holds explicit settings (usually from command-line flags) that take
// precedence over environment variables, which take precedence over the file.
type Overrides struct {
	ConfigPath            string
	Context               string
	Server                string
	Token                 string
	InsecureSkipTLSVerify bool
}

// Resolved is the effective connection configuration.
type Resolved struct {
	ConfigPath            string
	ContextName           string
	Server                string
	Token                 string
	InsecureSkipTLSVerify bool
}

// Resolve computes the effective settings: flags > environment > config file.
func Resolve(o Overrides) (*Resolved, error) {
	path := o.ConfigPath
	if path == "" {
		var err error
		if path, err = DefaultPath(); err != nil {
			return nil, err
		}
	}
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	r := &Resolved{ConfigPath: path, InsecureSkipTLSVerify: o.InsecureSkipTLSVerify}
	r.ContextName = firstNonEmpty(o.Context, os.Getenv(EnvContext), cfg.CurrentContext)
	if r.ContextName != "" {
		ctx, ok := cfg.Get(r.ContextName)
		switch {
		case ok:
			r.Server, r.Token = ctx.Server, ctx.Token
			r.InsecureSkipTLSVerify = r.InsecureSkipTLSVerify || ctx.InsecureSkipTLSVerify
		case o.Context != "" || os.Getenv(EnvContext) != "":
			return nil, fmt.Errorf("context %q not found in %s", r.ContextName, path)
		}
	}
	r.Server = firstNonEmpty(o.Server, os.Getenv(EnvServer), r.Server)
	r.Token = firstNonEmpty(o.Token, os.Getenv(EnvToken), r.Token)
	return r, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}
