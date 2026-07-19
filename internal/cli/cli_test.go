package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestRunRequiresExactlyOneFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run(nil, &stdout, &stderr); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "expected exactly one") {
		t.Fatalf("stderr did not explain the input contract: %q", stderr.String())
	}
}

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "ghosttag <image.jpg") {
		t.Fatalf("unexpected help: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunVersionUsesBuildOverride(t *testing.T) {
	previous := Version
	Version = "v1.2.3"
	t.Cleanup(func() { Version = previous })

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if stdout.String() != "ghosttag v1.2.3\n" {
		t.Fatalf("unexpected version output: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunReturnsFailureWhenHelpOrVersionCannotBeWritten(t *testing.T) {
	for _, argument := range []string{"--help", "--version"} {
		var stderr bytes.Buffer
		if code := Run([]string{argument}, failingWriter{}, &stderr); code != 1 {
			t.Errorf("%s exit code = %d, want 1", argument, code)
		}
		if !strings.Contains(stderr.String(), "write") {
			t.Errorf("%s did not report its writer failure: %q", argument, stderr.String())
		}
	}
}

func TestRunSanitizesUntrustedPathErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	path := "missing\n\x1b[31m\u202E.jpg"
	if code := Run([]string{path}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	message := stderr.String()
	if strings.ContainsRune(message, '\x1b') || strings.ContainsRune(message, '\u202E') || strings.Count(message, "\n") != 1 {
		t.Fatalf("unsafe diagnostic output: %q", message)
	}
}

var errCLIFailed = errors.New("writer failed")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errCLIFailed
}
