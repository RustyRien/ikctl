package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const currentUserQuery = `query CurrentUser {
  currentUser {
    id
    identifier
    displayName
    email
    provider
    entityName
  }
}`

const refreshAuthTokenMutation = `mutation RefreshAuthToken {
  refreshAuthToken {
    token
    expiration
    provider
  }
}`

const logoutMutation = `mutation Logout {
  logout {
    success
  }
}`

func (c *Client) CurrentUser(ctx context.Context) (*User, error) {
	resp, err := query[currentUserQueryData](ctx, c.httpClient, c.endpoint, c.tokenProvider, graphqlRequest{
		Query: currentUserQuery,
	})
	if err != nil {
		return nil, err
	}
	return resp.CurrentUser, nil
}

func (c *Client) RefreshAuthToken(ctx context.Context, provider string, refreshToken string) (*RefreshAuthTokenResult, error) {
	cookieName := refreshCookieName(provider)
	if cookieName == "" {
		return nil, fmt.Errorf("unsupported auth provider %q", provider)
	}
	if strings.TrimSpace(refreshToken) == "" {
		return nil, errors.New("refresh token is required")
	}

	endpointURL, err := url.Parse(c.endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse endpoint: %w", err)
	}
	c.httpClient.Jar.SetCookies(endpointURL, []*http.Cookie{{Name: cookieName, Value: refreshToken, Path: "/"}})

	resp, httpResp, err := queryWithHTTP[refreshAuthTokenMutationData](ctx, c.httpClient, c.endpoint, nil, graphqlRequest{Query: refreshAuthTokenMutation})
	if err != nil {
		return nil, err
	}
	if resp.RefreshAuthToken == nil {
		return nil, errors.New("refreshAuthToken returned no token")
	}
	resp.RefreshAuthToken.RefreshToken = extractRefreshToken(httpResp, cookieName)
	if resp.RefreshAuthToken.RefreshToken == "" {
		resp.RefreshAuthToken.RefreshToken = refreshToken
	}
	return resp.RefreshAuthToken, nil
}

func (c *Client) Logout(ctx context.Context, provider string, refreshToken string) error {
	cookieName := refreshCookieName(provider)
	if cookieName != "" && strings.TrimSpace(refreshToken) != "" {
		endpointURL, err := url.Parse(c.endpoint)
		if err != nil {
			return fmt.Errorf("parse endpoint: %w", err)
		}
		c.httpClient.Jar.SetCookies(endpointURL, []*http.Cookie{{Name: cookieName, Value: refreshToken, Path: "/"}})
	}
	resp, err := query[logoutMutationData](ctx, c.httpClient, c.endpoint, nil, graphqlRequest{Query: logoutMutation})
	if err != nil {
		return err
	}
	if resp.Logout == nil || !resp.Logout.Success {
		return errors.New("logout failed")
	}
	return nil
}

func refreshCookieName(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "microsoft":
		return "microsoft-refresh-token"
	case "github":
		return "github-refresh-token"
	case "guest":
		return "guest-token"
	default:
		return ""
	}
}

func extractRefreshToken(resp *http.Response, cookieName string) string {
	if resp == nil {
		return ""
	}
	for _, cookie := range resp.Cookies() {
		if cookie.Name == cookieName {
			return cookie.Value
		}
	}
	return ""
}
