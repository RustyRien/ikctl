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
	configRelPath          = "ik-tui/config.yaml"
)

type Config struct {
	Endpoint           string        `yaml:"endpoint"`
	Token              string        `yaml:"token"`
	RefreshInterval    time.Duration `yaml:"-"`
	RefreshSeconds     float64       `yaml:"refresh_seconds"`
	InsecureSkipVerify bool          `yaml:"insecure_skip_tls_verify"`
	ConfigPath         string        `yaml:"-"`
}

type fileConfig struct {
	Endpoint           string  `yaml:"endpoint"`
	Token              string  `yaml:"token"`
	RefreshSeconds     float64 `yaml:"refresh_seconds"`
	InsecureSkipVerify bool    `yaml:"insecure_skip_tls_verify"`
}

type FlagOverrides struct {
	ConfigPath         string
	Endpoint           string
	Token              string
	RefreshSeconds     float64
	InsecureSkipVerify bool
	HasInsecureFlag    bool
}

func Load(flags FlagOverrides) (Config, error) {
	cfg := Config{
		Endpoint:        defaultEndpoint,
		RefreshInterval: defaultRefreshInterval,
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
	cfg.InsecureSkipVerify = fileCfg.InsecureSkipVerify

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
}

func normalize(cfg *Config) {
	cfg.Endpoint = strings.TrimRight(cfg.Endpoint, "/")
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = defaultRefreshInterval
	}
	cfg.RefreshSeconds = cfg.RefreshInterval.Seconds()
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
