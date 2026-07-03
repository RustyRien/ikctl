package client

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/electrolux-oss/ik-tui/internal/config"
)

type Client struct {
	endpoint      string
	tokenProvider TokenProvider
	httpClient    *http.Client
}

type TokenProvider interface {
	Token(context.Context) (string, error)
}

type StaticToken string

func StaticTokenProvider(token string) TokenProvider {
	return StaticToken(token)
}

func (t StaticToken) Token(context.Context) (string, error) {
	return string(t), nil
}

func New(cfg config.Config) *Client {
	return NewWithTokenProvider(cfg, StaticTokenProvider(cfg.Token))
}

func NewWithTokenProvider(cfg config.Config, tokenProvider TokenProvider) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}
	jar, _ := cookiejar.New(nil)

	return &Client{
		endpoint:      cfg.Endpoint + "/api/graphql",
		tokenProvider: tokenProvider,
		httpClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
			Jar:       jar,
		},
	}
}
