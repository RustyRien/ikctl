package editor

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var editorEnvVars = []string{"IK_EDITOR", "K9S_EDITOR", "KUBE_EDITOR", "EDITOR"}

func EditFile(path string, stdin io.Reader, stdout, stderr *os.File) error {
	cmd, err := command(path)
	if err != nil {
		return err
	}
	if stdin == nil {
		stdin = os.Stdin
	}
	cmd.Stdin = stdin
	if stdout != nil {
		cmd.Stdout = stdout
	} else {
		cmd.Stdout = os.Stdout
	}
	if stderr != nil {
		cmd.Stderr = stderr
	} else {
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}

func command(path string) (*exec.Cmd, error) {
	var lastErr error
	for _, key := range editorEnvVars {
		raw := strings.TrimSpace(os.Getenv(key))
		if raw == "" {
			continue
		}
		tokens, err := splitCommand(raw)
		if err != nil || len(tokens) == 0 {
			lastErr = fmt.Errorf("parse %s: %w", key, err)
			continue
		}
		bin, err := exec.LookPath(tokens[0])
		if err != nil {
			lastErr = err
			continue
		}
		args := append(append([]string{}, tokens[1:]...), path)
		return exec.Command(bin, args...), nil
	}
	if lastErr != nil {
		return nil, fmt.Errorf("resolve editor: %w", lastErr)
	}
	bin, err := exec.LookPath("vi")
	if err == nil {
		return exec.Command(bin, path), nil
	}
	return nil, fmt.Errorf("no editor configured and vi not found; set %s", strings.Join(editorEnvVars, "|"))
}

func TempYAMLPath(kind string, name string) (string, error) {
	safeKind := sanitize(kind)
	safeName := sanitize(name)
	pattern := fmt.Sprintf("ikctl-edit-%s-%s-*.yaml", safeKind, safeName)
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		return "", errors.Join(err, os.Remove(path))
	}
	return filepath.Clean(path), nil
}

func splitCommand(input string) ([]string, error) {
	var (
		parts   []string
		current strings.Builder
		quote   rune
		escape  bool
	)
	flush := func() {
		if current.Len() == 0 {
			return
		}
		parts = append(parts, current.String())
		current.Reset()
	}
	for _, r := range input {
		switch {
		case escape:
			current.WriteRune(r)
			escape = false
		case r == '\\':
			escape = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t' || r == '\n':
			flush()
		default:
			current.WriteRune(r)
		}
	}
	if escape {
		return nil, fmt.Errorf("unterminated escape")
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	flush()
	return parts, nil
}

func sanitize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "entity"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}
