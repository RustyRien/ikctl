package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

const (
	defaultEndpoint        = "http://localhost:8000"
	defaultRefreshInterval = 2 * time.Second
	defaultSortOrder       = "desc"
	configRelPath          = "ikctl/config.yaml"
)

type Config struct {
	Endpoint           string        `yaml:"endpoint"`
	Token              string        `yaml:"token"`
	RefreshInterval    time.Duration `yaml:"-"`
	RefreshSeconds     float64       `yaml:"refresh_seconds"`
	AutoRefresh        bool          `yaml:"auto_refresh"`
	DefaultSortOrder   string        `yaml:"default_sort_order"`
	ShowBreadcrumbs    bool          `yaml:"show_breadcrumbs"`
	InsecureSkipVerify bool          `yaml:"insecure_skip_tls_verify"`
	NoColors           bool          `yaml:"no_colors"`
	View               ViewConfig    `yaml:"-"`
	ConfigPath         string        `yaml:"-"`
}

type ViewConfig struct {
	Resources ResourceViewConfig `yaml:"resources,omitempty"`
	Templates TemplateViewConfig `yaml:"templates,omitempty"`
}

type ResourceViewConfig struct {
	Columns                 []string  `yaml:"columns,omitempty"`
	StorageFilter           FilterRef `yaml:"storage_filter,omitempty"`
	SecretFilter            FilterRef `yaml:"secret_filter,omitempty"`
	TemplateFilter          FilterRef `yaml:"template_filter,omitempty"`
	SourceCodeVersionFilter FilterRef `yaml:"source_code_version_filter,omitempty"`
	IntegrationFilter       FilterRef `yaml:"integration_filter,omitempty"`
	HideDestroyed           bool      `yaml:"hide_destroyed,omitempty"`
}

type TemplateViewConfig struct {
	Columns []string `yaml:"columns,omitempty"`
}

type FilterRef struct {
	ID   string `yaml:"id,omitempty"`
	Name string `yaml:"name,omitempty"`
}

type fileConfig struct {
	Endpoint           string      `yaml:"endpoint"`
	Token              string      `yaml:"token"`
	RefreshSeconds     float64     `yaml:"refresh_seconds"`
	AutoRefresh        *bool       `yaml:"auto_refresh,omitempty"`
	DefaultSortOrder   string      `yaml:"default_sort_order,omitempty"`
	ShowBreadcrumbs    *bool       `yaml:"show_breadcrumbs,omitempty"`
	InsecureSkipVerify bool        `yaml:"insecure_skip_tls_verify"`
	NoColors           bool        `yaml:"no_colors"`
	TUI                *ViewConfig `yaml:"tui,omitempty"`
}

type FlagOverrides struct {
	ConfigPath         string
	Endpoint           string
	Token              string
	RefreshSeconds     float64
	InsecureSkipVerify bool
	HasInsecureFlag    bool
	NoColors           bool
	HasNoColorsFlag    bool
}

func Load(flags FlagOverrides) (Config, error) {
	cfg := Config{
		Endpoint:         defaultEndpoint,
		RefreshInterval:  defaultRefreshInterval,
		AutoRefresh:      true,
		DefaultSortOrder: defaultSortOrder,
		ShowBreadcrumbs:  true,
	}

	configPath, err := resolveConfigPath(flags.ConfigPath)
	if err != nil {
		return Config{}, err
	}
	cfg.ConfigPath = configPath

	if err := loadFile(&cfg, configPath); err != nil {
		return Config{}, err
	}

	applyEnv(&cfg)
	applyFlags(&cfg, flags)
	normalize(&cfg)

	if cfg.Endpoint == "" {
		return Config{}, errors.New("endpoint is required")
	}
	if cfg.DefaultSortOrder != "asc" && cfg.DefaultSortOrder != "desc" {
		return Config{}, fmt.Errorf("default sort order must be asc or desc, got %q", cfg.DefaultSortOrder)
	}

	return cfg, nil
}

func DefaultConfigPath() (string, error) {
	return resolveConfigPath("")
}

func resolveConfigPath(override string) (string, error) {
	if override != "" {
		return override, nil
	}

	path, err := xdg.ConfigFile(configRelPath)
	if err != nil {
		return "", fmt.Errorf("resolve config path: %w", err)
	}

	return path, nil
}

func loadFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read config file: %w", err)
	}

	var fileCfg fileConfig
	if err := yaml.Unmarshal(data, &fileCfg); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}

	if fileCfg.Endpoint != "" {
		cfg.Endpoint = fileCfg.Endpoint
	}
	if fileCfg.Token != "" {
		cfg.Token = fileCfg.Token
	}
	if fileCfg.RefreshSeconds > 0 {
		cfg.RefreshSeconds = fileCfg.RefreshSeconds
		cfg.RefreshInterval = time.Duration(fileCfg.RefreshSeconds * float64(time.Second))
	}
	if fileCfg.AutoRefresh != nil {
		cfg.AutoRefresh = *fileCfg.AutoRefresh
	}
	if fileCfg.DefaultSortOrder != "" {
		cfg.DefaultSortOrder = fileCfg.DefaultSortOrder
	}
	if fileCfg.ShowBreadcrumbs != nil {
		cfg.ShowBreadcrumbs = *fileCfg.ShowBreadcrumbs
	}
	cfg.InsecureSkipVerify = fileCfg.InsecureSkipVerify
	cfg.NoColors = fileCfg.NoColors
	if fileCfg.TUI != nil {
		cfg.View = *fileCfg.TUI
	}

	return nil
}

func applyEnv(cfg *Config) {
	if value := os.Getenv("IK_ENDPOINT"); value != "" {
		cfg.Endpoint = value
	}
	if value := os.Getenv("IK_TOKEN"); value != "" {
		cfg.Token = value
	}
	if value := os.Getenv("IK_REFRESH_SECONDS"); value != "" {
		if d, err := parseSeconds(value); err == nil {
			cfg.RefreshInterval = d
		}
	}
	if value := os.Getenv("IK_INSECURE_SKIP_TLS_VERIFY"); value != "" {
		cfg.InsecureSkipVerify = strings.EqualFold(value, "true") || value == "1"
	}
	if value := os.Getenv("IK_NO_COLORS"); value != "" {
		cfg.NoColors = strings.EqualFold(value, "true") || value == "1"
	}
	if value := os.Getenv("IK_AUTO_REFRESH"); value != "" {
		cfg.AutoRefresh = strings.EqualFold(value, "true") || value == "1"
	}
	if value := os.Getenv("IK_DEFAULT_SORT_ORDER"); value != "" {
		cfg.DefaultSortOrder = value
	}
	if value := os.Getenv("IK_SHOW_BREADCRUMBS"); value != "" {
		cfg.ShowBreadcrumbs = strings.EqualFold(value, "true") || value == "1"
	}
}

func applyFlags(cfg *Config, flags FlagOverrides) {
	if flags.Endpoint != "" {
		cfg.Endpoint = flags.Endpoint
	}
	if flags.Token != "" {
		cfg.Token = flags.Token
	}
	if flags.RefreshSeconds > 0 {
		cfg.RefreshInterval = time.Duration(flags.RefreshSeconds * float64(time.Second))
	}
	if flags.HasInsecureFlag || flags.InsecureSkipVerify {
		cfg.InsecureSkipVerify = flags.InsecureSkipVerify
	}
	if flags.HasNoColorsFlag || flags.NoColors {
		cfg.NoColors = flags.NoColors
	}
}

func normalize(cfg *Config) {
	cfg.Endpoint = strings.TrimRight(cfg.Endpoint, "/")
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = defaultRefreshInterval
	}
	cfg.RefreshSeconds = cfg.RefreshInterval.Seconds()
	cfg.DefaultSortOrder = strings.ToLower(strings.TrimSpace(cfg.DefaultSortOrder))
	if cfg.DefaultSortOrder == "" {
		cfg.DefaultSortOrder = defaultSortOrder
	}
	if cfg.ConfigPath != "" {
		cfg.ConfigPath = filepath.Clean(cfg.ConfigPath)
	}
}

func parseSeconds(value string) (time.Duration, error) {
	seconds, err := time.ParseDuration(value + "s")
	if err != nil {
		return 0, err
	}
	return seconds, nil
}

func (c *Config) Save() error {
	if c.ConfigPath == "" {
		return errors.New("cannot save: config path is empty")
	}

	dir := filepath.Dir(c.ConfigPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	fileCfg := fileConfig{
		Endpoint:           c.Endpoint,
		Token:              c.Token,
		RefreshSeconds:     c.RefreshSeconds,
		AutoRefresh:        &c.AutoRefresh,
		DefaultSortOrder:   c.DefaultSortOrder,
		ShowBreadcrumbs:    &c.ShowBreadcrumbs,
		InsecureSkipVerify: c.InsecureSkipVerify,
		NoColors:           c.NoColors,
	}
	if !c.View.Empty() {
		view := c.View
		fileCfg.TUI = &view
	}

	data, err := yaml.Marshal(fileCfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(c.ConfigPath, data, 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

func (c Config) DefaultSortDescending() bool {
	return c.DefaultSortOrder != "asc"
}

func (v ViewConfig) Empty() bool {
	return v.Resources.Empty() && v.Templates.Empty()
}

func (v ResourceViewConfig) Empty() bool {
	return len(v.Columns) == 0 && v.StorageFilter.Empty() && v.SecretFilter.Empty() && v.TemplateFilter.Empty() && v.SourceCodeVersionFilter.Empty() && v.IntegrationFilter.Empty() && !v.HideDestroyed
}

func (v TemplateViewConfig) Empty() bool {
	return len(v.Columns) == 0
}

func (v FilterRef) Empty() bool {
	return v.ID == "" && v.Name == ""
}
