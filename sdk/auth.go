package sdk

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// TokenPrefix is the prefix of InfraKitchen personal access tokens.
const TokenPrefix = "ik_"

var tokenIDRe = regexp.MustCompile(`^[0-9a-f]{32}$`)

// TokenInfo describes a parsed personal access token. It never contains the secret.
type TokenInfo struct {
	// ID is the token id (32 hex characters).
	ID string `json:"id"`
	// Prefix is the public display prefix, e.g. "ik_55d15ded".
	Prefix string `json:"prefix"`
}

// IsPersonalAccessToken reports whether token looks like a personal access
// token (as opposed to a service-account JWT).
func IsPersonalAccessToken(token string) bool { return strings.HasPrefix(token, TokenPrefix) }

// ParseToken validates the format of a personal access token
// ("ik_<32 hex>_<secret>"), splitting it the same way the server does.
func ParseToken(token string) (TokenInfo, error) {
	token = strings.TrimSpace(token)
	parts := strings.SplitN(token, "_", 3)
	if len(parts) != 3 || parts[0]+"_" != TokenPrefix {
		return TokenInfo{}, fmt.Errorf("invalid token format: expected %s<id>_<secret>", TokenPrefix)
	}
	if !tokenIDRe.MatchString(parts[1]) {
		return TokenInfo{}, fmt.Errorf("invalid token format: token id must be 32 lowercase hex characters")
	}
	if parts[2] == "" {
		return TokenInfo{}, fmt.Errorf("invalid token format: missing secret")
	}
	return TokenInfo{ID: parts[1], Prefix: TokenPrefix + parts[1][:8]}, nil
}

// RedactToken returns a display-safe form of a token.
func RedactToken(token string) string {
	if token == "" {
		return ""
	}
	if info, err := ParseToken(token); err == nil {
		return info.Prefix + "...REDACTED"
	}
	return "REDACTED"
}

// User is the authenticated InfraKitchen user.
type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Identifier  string    `json:"identifier"`
	FirstName   string    `json:"first_name,omitempty"`
	LastName    string    `json:"last_name,omitempty"`
	DisplayName string    `json:"display_name"`
	Provider    string    `json:"provider"`
	Deactivated bool      `json:"deactivated"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AuthService provides authentication operations.
type AuthService struct{ c *Client }

// Me returns the user that owns the configured token. It does not depend on
// token scopes, so it is the recommended way to validate credentials.
func (s *AuthService) Me(ctx context.Context) (*User, error) {
	if s.c.token == "" {
		return nil, &APIError{Code: CodeUnauthorized, Message: "no token configured: run 'ikctl login' or set IKCTL_TOKEN"}
	}
	body, err := s.c.doHTTP(ctx, http.MethodGet, MePath, nil)
	if err != nil {
		return nil, err
	}
	var u User
	if err := decodeJSON(body, &u); err != nil {
		return nil, err
	}
	if u.ID == "" {
		return nil, &APIError{Code: CodeUnauthorized, Message: "server did not return a user for this token"}
	}
	return &u, nil
}

// ValidateToken checks the configured token against the server.
func (s *AuthService) ValidateToken(ctx context.Context) error {
	_, err := s.Me(ctx)
	return err
}

// ServerInfo describes the InfraKitchen server build.
type ServerInfo struct {
	Version           string `json:"version"`
	VersionURL        string `json:"versionUrl,omitempty"`
	Repository        string `json:"repository,omitempty"`
	RepositoryURL     string `json:"repositoryUrl,omitempty"`
	SourceCommit      string `json:"sourceCommit,omitempty"`
	SourceCommitShort string `json:"sourceCommitShort,omitempty"`
	SourceURL         string `json:"sourceUrl,omitempty"`
}

// ServerService provides server-level information.
type ServerService struct{ c *Client }

// Info returns the server version information.
func (s *ServerService) Info(ctx context.Context) (*ServerInfo, error) {
	const q = `query ServerInfo { serverInfo { version versionUrl repository repositoryUrl sourceCommit sourceCommitShort sourceUrl } }`
	var data struct {
		ServerInfo ServerInfo `json:"serverInfo"`
	}
	if err := s.c.Do(ctx, Request{Query: q}, &data); err != nil {
		return nil, err
	}
	return &data.ServerInfo, nil
}

// EntityNames returns the entity names known by the server.
func (s *ServerService) EntityNames(ctx context.Context) ([]string, error) {
	var data struct {
		Entities []string `json:"entities"`
	}
	if err := s.c.Do(ctx, Request{Query: `query { entities }`}, &data); err != nil {
		return nil, err
	}
	return data.Entities, nil
}
