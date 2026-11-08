// Package logging provides a privacy-aware slog logger that writes to
// the user's ~/.config/osaat/logs/ directory and scrubs sensitive
// paths from every record.
//
// The scrub policy:
//   - $HOME is replaced with "~" in messages and string attribute values.
//   - Hostname is never logged (no telemetry that could identify a machine).
//   - License keys and secret values must never be passed to the logger;
//     enforce that at call sites — this package can't introspect what
//     callers send.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/simtabi/osaat/internal/paths"
)

// NewFileLogger creates a daily log file under logDir (mode 600) and
// returns a slog.Logger that writes to it. The returned Closer should
// be closed at program exit.
//
// The path is logDir/osaat-YYYY-MM-DD.log. If a file for today
// already exists, new records are appended.
func NewFileLogger(logDir string, level slog.Level) (*slog.Logger, io.Closer, error) {
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return nil, nil, fmt.Errorf("create log dir: %w", err)
	}
	filename := filepath.Join(logDir, fmt.Sprintf("osaat-%s.log", time.Now().Format("2006-01-02")))
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}
	h := NewScrubHandler(f, &slog.HandlerOptions{Level: level})
	return slog.New(h), f, nil
}

// NewTeeLogger returns a logger that fans every record out to all of
// the provided slog.Handlers. Used to write both to stderr and to a
// file.
func NewTeeLogger(handlers ...slog.Handler) *slog.Logger {
	return slog.New(&teeHandler{handlers: handlers})
}

type teeHandler struct {
	handlers []slog.Handler
}

func (t *teeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range t.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (t *teeHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, h := range t.handlers {
		if err := h.Handle(ctx, r.Clone()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (t *teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, len(t.handlers))
	for i, h := range t.handlers {
		next[i] = h.WithAttrs(attrs)
	}
	return &teeHandler{handlers: next}
}

func (t *teeHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, len(t.handlers))
	for i, h := range t.handlers {
		next[i] = h.WithGroup(name)
	}
	return &teeHandler{handlers: next}
}

// NewScrubHandler wraps a slog.TextHandler and replaces $HOME with ~
// in every string attribute and in the message.
func NewScrubHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &scrubHandler{inner: slog.NewTextHandler(w, opts)}
}

type scrubHandler struct {
	inner slog.Handler
}

func (s *scrubHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return s.inner.Enabled(ctx, level)
}

func (s *scrubHandler) Handle(ctx context.Context, r slog.Record) error {
	clone := slog.NewRecord(r.Time, r.Level, paths.ScrubHome(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		clone.AddAttrs(s.scrubAttr(a))
		return true
	})
	return s.inner.Handle(ctx, clone)
}

func (s *scrubHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	scrubbed := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		scrubbed[i] = s.scrubAttr(a)
	}
	return &scrubHandler{inner: s.inner.WithAttrs(scrubbed)}
}

func (s *scrubHandler) WithGroup(name string) slog.Handler {
	return &scrubHandler{inner: s.inner.WithGroup(name)}
}

func (s *scrubHandler) scrubAttr(a slog.Attr) slog.Attr {
	switch a.Value.Kind() {
	case slog.KindString:
		a.Value = slog.StringValue(paths.ScrubHome(a.Value.String()))
	case slog.KindGroup:
		grp := a.Value.Group()
		scrubbed := make([]slog.Attr, len(grp))
		for i, sub := range grp {
			scrubbed[i] = s.scrubAttr(sub)
		}
		a.Value = slog.GroupValue(scrubbed...)
	case slog.KindAny:
		if str, ok := a.Value.Any().(string); ok {
			a.Value = slog.StringValue(paths.ScrubHome(str))
		}
	}
	// strip hostname-ish keys defensively
	if isHostnameKey(a.Key) {
		a.Value = slog.StringValue("[redacted]")
	}
	return a
}

func isHostnameKey(key string) bool {
	switch strings.ToLower(key) {
	case "hostname", "host", "machine", "node":
		return true
	}
	return false
}
