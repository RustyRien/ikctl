package edit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/electrolux-oss/ik-tui/internal/editor"
	"github.com/electrolux-oss/ik-tui/internal/printer"
	"gopkg.in/yaml.v3"
)

var ErrEditCancelled = errors.New("edit cancelled: no changes made")

var openEditor = editor.EditFile

type Session struct {
	Kind  string
	ID    string
	Name  string
	Load  func(context.Context, string) ([]byte, error)
	Apply func(context.Context, string, []byte) error
}

func Run(ctx context.Context, session Session, stdin io.Reader, stdout, stderr *os.File) error {
	return RunPhased(ctx, ctx, session, stdin, stdout, stderr)
}

func RunPhased(loadCtx context.Context, applyCtx context.Context, session Session, stdin io.Reader, stdout, stderr *os.File) error {
	if session.Load == nil || session.Apply == nil {
		return fmt.Errorf("edit is not supported for %s", session.Kind)
	}
	content, err := session.Load(loadCtx, session.ID)
	if err != nil {
		return err
	}

	path, err := PrepareSessionFile(session.Kind, session.Name, content)
	if err != nil {
		return err
	}
	defer os.Remove(path)
	before, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := OpenEditor(path, stdin, stdout, stderr); err != nil {
		return err
	}
	after, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if bytes.Equal(before, after) {
		return ErrEditCancelled
	}
	if err := session.Apply(applyCtx, session.ID, after); err != nil {
		return err
	}
	return nil
}

func PrepareSessionFile(kind string, name string, content []byte) (string, error) {
	path, err := editor.TempYAMLPath(kind, name)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func OpenEditor(path string, stdin io.Reader, stdout, stderr *os.File) error {
	return openEditor(path, stdin, stdout, stderr)
}

func YAMLBytes(raw any) ([]byte, error) {
	var buf bytes.Buffer
	if err := printer.Print(&buf, "yaml", nil, nil, []any{raw}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ParseYAMLMap(data []byte) (map[string]any, error) {
	var decoded map[string]any
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	if decoded == nil {
		return nil, fmt.Errorf("empty YAML document")
	}
	return decoded, nil
}
