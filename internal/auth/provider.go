package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/config"
)

const tokenRefreshLeeway = 30 * time.Second

type Manager struct {
	endpoint string
	store    *Store
	client   *client.Client
	mu       sync.Mutex
	entry    Credentials
	loaded   bool
	refresh  *refreshState
}

type refreshState struct {
	done  chan struct{}
	token string
	err   error
}

func NewClient(cfg config.Config) (*client.Client, error) {
	provider, err := NewTokenProvider(cfg)
	if err != nil {
		return nil, err
	}
	return client.NewWithTokenProvider(cfg, provider), nil
}

func NewTokenProvider(cfg config.Config) (client.TokenProvider, error) {
	if strings.TrimSpace(cfg.Token) != "" {
		return client.StaticTokenProvider(cfg.Token), nil
	}

	store, err := OpenStore("")
	if err != nil {
		return nil, err
	}

	manager := &Manager{
		endpoint: cfg.Endpoint,
		store:    store,
		client:   client.NewWithTokenProvider(cfg, nil),
	}
	return manager, nil
}

func (m *Manager) Token(ctx context.Context) (string, error) {
	m.mu.Lock()
	if !m.loaded {
		entry, ok := m.store.Get(m.endpoint)
		if ok {
			m.entry = entry
		}
		m.loaded = true
	}

	if token := strings.TrimSpace(m.entry.Token); token != "" && m.entry.TokenExpiry.After(time.Now().Add(tokenRefreshLeeway)) {
		m.mu.Unlock()
		return token, nil
	}

	if strings.TrimSpace(m.entry.RefreshToken) == "" || strings.TrimSpace(m.entry.Provider) == "" {
		m.mu.Unlock()
		return "", nil
	}

	if m.refresh != nil {
		refresh := m.refresh
		m.mu.Unlock()
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-refresh.done:
			return refresh.token, refresh.err
		}
	}

	refresh := &refreshState{done: make(chan struct{})}
	m.refresh = refresh
	entry := m.entry
	m.mu.Unlock()

	result, err := m.client.RefreshAuthToken(ctx, entry.Provider, entry.RefreshToken)
	if err == nil {
		entry.Token = result.Token
		entry.TokenExpiry = result.Expiration.Time
		if strings.TrimSpace(result.RefreshToken) != "" {
			entry.RefreshToken = result.RefreshToken
		}
		err = m.store.Put(m.endpoint, entry)
	}

	m.mu.Lock()
	if err == nil {
		m.entry = entry
		refresh.token = entry.Token
	} else {
		refresh.token = ""
	}
	refresh.err = err
	m.refresh = nil
	close(refresh.done)
	m.mu.Unlock()

	if err != nil {
		return "", err
	}
	return entry.Token, nil
}

func ParseProvider(value string) (string, string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "microsoft", "github":
		return strings.ToLower(strings.TrimSpace(value)), "", nil
	case "guest", "guest_default":
		return "guest", "default", nil
	case "guest_super":
		return "guest", "super", nil
	case "guest_infra":
		return "guest", "infra", nil
	default:
		return "", "", fmt.Errorf("unsupported provider %q", value)
	}
}

func ValidateGuestScope(scope string) error {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "default", "super", "infra":
		return nil
	default:
		return errors.New("guest scope must be one of: default, super, infra")
	}
}

func LoginPath(provider string, scope string) string {
	if provider == "guest" {
		return "/api/auth/guest/login/" + scope
	}
	return "/api/auth/" + provider + "/login"
}
