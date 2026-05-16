package macos

import (
	"context"
	"strings"

	"github.com/simtabi/osaat/internal/audit"
)

// enrichFromQuarantine reads the `com.apple.quarantine` extended
// attribute on each app. Presence of the attribute means the app went
// through Gatekeeper, which is itself a signal that the app came from
// the internet (vs. App Store, system, or a package manager).
//
// The attribute encodes flags; agent; timestamp; uuid — we don't try
// to look up the URL via the LaunchServices LSQuarantineDataURLKey
// from the CLI, which is gated. We instead record the agent (e.g.
// "Safari", "Chrome") as a CollectorNote.
func (c *Collector) enrichFromQuarantine(ctx context.Context, records []audit.AppRecord) error {
	for i := range records {
		if records[i].Path == "" {
			continue
		}
		out, err := c.runCmd(ctx, "xattr", "-p", "com.apple.quarantine", records[i].Path)
		if err != nil {
			continue
		}
		agent := parseQuarantineAgent(out)
		if agent != "" {
			records[i].Note("quarantined by " + agent)
			if records[i].Source == audit.SourceUnknown {
				records[i].Source = audit.SourceDMG
			}
		}
	}
	return nil
}

// parseQuarantineAgent extracts the agent field from an xattr value of
// the form "0001;6537e2a3;Safari;UUID". Empty if absent or malformed.
func parseQuarantineAgent(out []byte) string {
	parts := strings.Split(strings.TrimSpace(string(out)), ";")
	if len(parts) < 3 {
		return ""
	}
	return strings.TrimSpace(parts[2])
}
