package logging

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScrubHandlerReplacesHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir on this runner")
	}
	var buf bytes.Buffer
	logger := slog.New(NewScrubHandler(&buf, nil))
	logger.Info("opened", "path", filepath.Join(home, "Library", "Preferences", "x.plist"))

	out := buf.String()
	if strings.Contains(out, home) {
		t.Errorf("home leaked: %q", out)
	}
	if !strings.Contains(out, "~") {
		t.Errorf("expected ~ in output: %q", out)
	}
}

func TestScrubHandlerStripsHostname(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewScrubHandler(&buf, nil))
	logger.Info("ping", "hostname", "Imanis-MacBook-Pro.local", "msg", "x")

	out := buf.String()
	if strings.Contains(out, "Imanis") {
		t.Errorf("hostname leaked: %q", out)
	}
	if !strings.Contains(out, "[redacted]") {
		t.Errorf("expected redaction marker: %q", out)
	}
}

func TestNewFileLoggerCreatesFile(t *testing.T) {
	dir := t.TempDir()
	logger, closer, err := NewFileLogger(dir, slog.LevelInfo)
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("hello world", "k", "v")
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 log file; got %v", entries)
	}
	if !strings.HasPrefix(entries[0].Name(), "osaat-") {
		t.Errorf("filename should start with osaat-; got %q", entries[0].Name())
	}
	info, err := entries[0].Info()
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("log mode should be 600; got %o", perm)
	}
}
