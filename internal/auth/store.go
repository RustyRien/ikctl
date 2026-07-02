package auth

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

const credentialsRelPath = "ikctl/credentials.yaml"

type Credentials struct {
	Provider     string    `yaml:"provider"`
	RefreshToken string    `yaml:"refresh_token"`
	Scope        string    `yaml:"scope,omitempty"`
	Token        string    `yaml:"token,omitempty"`
	TokenExpiry  time.Time `yaml:"token_expiry,omitempty"`
}

type credentialsFile struct {
	Endpoints map[string]Credentials `yaml:"endpoints"`
}

type Store struct {
	path string
	mu   sync.Mutex
	data credentialsFile
}

func DefaultCredentialsPath() (string, error) {
	path, err := xdg.ConfigFile(credentialsRelPath)
	if err != nil {
		return "", fmt.Errorf("resolve credentials path: %w", err)
	}
	return filepath.Clean(path), nil
}

func OpenStore(path string) (*Store, error) {
	if path == "" {
		var err error
		path, err = DefaultCredentialsPath()
		if err != nil {
			return nil, err
		}
	}

	store := &Store{path: filepath.Clean(path), data: credentialsFile{Endpoints: map[string]Credentials{}}}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Get(endpoint string) (Credentials, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cred, ok := s.data.Endpoints[endpoint]
	return cred, ok
}

func (s *Store) Put(endpoint string, cred Credentials) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Endpoints == nil {
		s.data.Endpoints = map[string]Credentials{}
	}
	s.data.Endpoints[endpoint] = cred
	return s.saveLocked()
}

func (s *Store) Delete(endpoint string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Endpoints, endpoint)
	return s.saveLocked()
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read credentials file: %w", err)
	}

	var parsed credentialsFile
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("parse credentials file: %w", err)
	}
	if parsed.Endpoints == nil {
		parsed.Endpoints = map[string]Credentials{}
	}
	s.data = parsed
	return nil
}

func (s *Store) saveLocked() error {
	data, err := yaml.Marshal(s.data)
	if err != nil {
		return fmt.Errorf("marshal credentials file: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create credentials directory: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write credentials file: %w", err)
	}
	return nil
}
