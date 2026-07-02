package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/coder/websocket"
)

const logStreamSubscription = `subscription LogStream($entityName: String!, $entityId: String!) {
  logStream(entityName: $entityName, entityId: $entityId) {
    data
    level
  }
}`

type wsConnectionInit struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload,omitempty"`
}

type wsSubscribe struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Payload graphqlRequest `json:"payload"`
}

type wsMessage struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type wsNextPayload struct {
	Data   logStreamData  `json:"data"`
	Errors []GraphQLError `json:"errors"`
}

type logStreamData struct {
	LogStream *LogStreamMessage `json:"logStream"`
}

func (c *Client) StreamLogs(ctx context.Context, entityName string, entityID string, onMessage func(LogStreamMessage) error) error {
	if strings.TrimSpace(entityName) == "" {
		return errors.New("entity name is required")
	}
	if strings.TrimSpace(entityID) == "" {
		return errors.New("entity ID is required")
	}
	if onMessage == nil {
		return errors.New("message callback is required")
	}

	endpoint, err := websocketEndpoint(c.endpoint)
	if err != nil {
		return err
	}

	httpClient := c.httpClient
	transport, _ := httpClient.Transport.(*http.Transport)
	dialOptions := &websocket.DialOptions{
		Subprotocols: []string{"graphql-transport-ws"},
	}
	if transport != nil && transport.TLSClientConfig != nil {
		dialOptions.HTTPClient = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: transport.TLSClientConfig.InsecureSkipVerify}}}
	}

	conn, _, err := websocket.Dial(ctx, endpoint, dialOptions)
	if err != nil {
		return fmt.Errorf("dial graphql websocket: %w", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	token := ""
	if c.tokenProvider != nil {
		token, err = c.tokenProvider.Token(ctx)
		if err != nil {
			return fmt.Errorf("resolve bearer token: %w", err)
		}
	}

	if err := writeWS(ctx, conn, wsConnectionInit{
		Type: "connection_init",
		Payload: map[string]any{
			"token": token,
		},
	}); err != nil {
		return err
	}

	if err := awaitWSAck(ctx, conn); err != nil {
		return err
	}

	const operationID = "log-stream"
	if err := writeWS(ctx, conn, wsSubscribe{
		ID:   operationID,
		Type: "subscribe",
		Payload: graphqlRequest{
			Query: logStreamSubscription,
			Variables: map[string]any{
				"entityName": entityName,
				"entityId":   entityID,
			},
		},
	}); err != nil {
		return err
	}

	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			_ = writeWS(context.Background(), conn, wsMessage{ID: operationID, Type: "complete"})
		})
	}
	defer stop()

	go func() {
		<-ctx.Done()
		stop()
		_ = conn.Close(websocket.StatusNormalClosure, "context canceled")
	}()

	for {
		var message wsMessage
		if err := readWS(ctx, conn, &message); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			closeErr := websocket.CloseStatus(err)
			if closeErr == websocket.StatusNormalClosure || closeErr == websocket.StatusGoingAway {
				return nil
			}
			return fmt.Errorf("read graphql websocket message: %w", err)
		}

		switch message.Type {
		case "next":
			var payload wsNextPayload
			if err := json.Unmarshal(message.Payload, &payload); err != nil {
				return fmt.Errorf("decode graphql subscription payload: %w", err)
			}
			if len(payload.Errors) > 0 {
				messages := make([]string, 0, len(payload.Errors))
				for _, gqlErr := range payload.Errors {
					messages = append(messages, gqlErr.Message)
				}
				return errors.New(strings.Join(messages, "; "))
			}
			if payload.Data.LogStream == nil {
				continue
			}
			if err := onMessage(*payload.Data.LogStream); err != nil {
				return err
			}
		case "complete":
			return nil
		case "error":
			var gqlErrs []GraphQLError
			if len(message.Payload) > 0 && json.Unmarshal(message.Payload, &gqlErrs) == nil && len(gqlErrs) > 0 {
				messages := make([]string, 0, len(gqlErrs))
				for _, gqlErr := range gqlErrs {
					messages = append(messages, gqlErr.Message)
				}
				return errors.New(strings.Join(messages, "; "))
			}
			return errors.New("graphql subscription returned an error")
		case "ping":
			if err := writeWS(ctx, conn, wsMessage{Type: "pong"}); err != nil {
				return err
			}
		case "pong":
		case "ka":
		default:
		}
	}
}

func websocketEndpoint(endpoint string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse graphql endpoint: %w", err)
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported endpoint scheme %q", parsed.Scheme)
	}
	return parsed.String(), nil
}

func awaitWSAck(ctx context.Context, conn *websocket.Conn) error {
	for {
		var message wsMessage
		if err := readWS(ctx, conn, &message); err != nil {
			return fmt.Errorf("await graphql websocket ack: %w", err)
		}
		switch message.Type {
		case "connection_ack":
			return nil
		case "connection_error":
			return errors.New("graphql websocket connection rejected")
		case "ping":
			if err := writeWS(ctx, conn, wsMessage{Type: "pong"}); err != nil {
				return err
			}
		case "pong":
		case "ka":
		default:
		}
	}
}

func writeWS(ctx context.Context, conn *websocket.Conn, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode graphql websocket message: %w", err)
	}
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		return fmt.Errorf("write graphql websocket message: %w", err)
	}
	return nil
}

func readWS(ctx context.Context, conn *websocket.Conn, target any) error {
	_, data, err := conn.Read(ctx)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode graphql websocket message: %w", err)
	}
	return nil
}
