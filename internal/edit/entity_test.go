package edit

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

func TestRunPhasedAppliesAfterEditorDelay(t *testing.T) {
	oldOpenEditor := openEditor
	t.Cleanup(func() { openEditor = oldOpenEditor })

	openEditor = func(path string, stdin io.Reader, stdout, stderr *os.File) error {
		time.Sleep(25 * time.Millisecond)
		return os.WriteFile(path, []byte("name: updated\n"), 0o600)
	}

	loadCtx, loadCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer loadCancel()
	applyCtx, applyCancel := context.WithTimeout(context.Background(), time.Second)
	defer applyCancel()

	applied := false
	err := RunPhased(loadCtx, applyCtx, Session{
		Kind: "resource",
		ID:   "r1",
		Name: "redis",
		Load: func(ctx context.Context, id string) ([]byte, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return []byte("name: original\n"), nil
		},
		Apply: func(ctx context.Context, id string, data []byte) error {
			applied = true
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if string(data) != "name: updated\n" {
				t.Fatalf("unexpected apply data %q", string(data))
			}
			return nil
		},
	}, nil, nil, nil)
	if err != nil {
		t.Fatalf("RunPhased: %v", err)
	}
	if !applied {
		t.Fatal("expected Apply to be called")
	}
}

func TestRunPhasedNoChangesCancels(t *testing.T) {
	oldOpenEditor := openEditor
	t.Cleanup(func() { openEditor = oldOpenEditor })

	openEditor = func(path string, stdin io.Reader, stdout, stderr *os.File) error {
		return nil
	}

	err := RunPhased(context.Background(), context.Background(), Session{
		Kind: "resource",
		ID:   "r1",
		Name: "redis",
		Load: func(ctx context.Context, id string) ([]byte, error) {
			return []byte("name: original\n"), nil
		},
		Apply: func(ctx context.Context, id string, data []byte) error {
			t.Fatal("Apply should not be called")
			return nil
		},
	}, nil, nil, nil)
	if !errors.Is(err, ErrEditCancelled) {
		t.Fatalf("expected ErrEditCancelled, got %v", err)
	}
}
