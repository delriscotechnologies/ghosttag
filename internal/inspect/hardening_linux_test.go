//go:build linux

package inspect

import (
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRejectsFIFOWithoutBlocking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image.png")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := File(path)
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "regular file") {
			t.Fatalf("expected regular-file error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("inspection blocked while opening a FIFO")
	}
}
