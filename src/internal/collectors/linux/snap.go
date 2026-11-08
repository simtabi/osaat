package linux

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/simtabi/osaat/internal/audit"
)

// collectSnap runs `snap list --color=never --unicode=never` and
// returns one AppRecord per installed snap. The flags request
// machine-friendly output: no ANSI colors, no Unicode characters
// like ✓ next to verified publishers.
func (c *Collector) collectSnap(ctx context.Context) ([]audit.AppRecord, error) {
	out, err := c.runCmd(ctx, "snap", "list", "--color=never", "--unicode=never")
	if err != nil {
		return nil, fmt.Errorf("snap list: %w", err)
	}
	return parseSnapOutput(out), nil
}

// parseSnapOutput reads `snap list` output and emits records. The
// header row is detected by the first column being literally "Name".
// We tolerate the Unicode ✓ checkmark next to publisher names so the
// parser works even when --unicode=never isn't honored (older snapd
// versions).
//
// Columns: Name  Version  Rev  Tracking  Publisher  Notes
// "Notes" can be a single word or empty (rendered as "-").
func parseSnapOutput(out []byte) []audit.AppRecord {
	var records []audit.AppRecord
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)

	first := true
	for sc.Scan() {
		raw := sc.Text()
		if first {
			first = false
			if strings.HasPrefix(strings.TrimSpace(raw), "Name") {
				continue // header
			}
		}
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		name := fields[0]
		version := fields[1]
		// fields[2] = rev (skip)
		// fields[3] = tracking channel
		channel := fields[3]
		publisher := stripVerifiedMark(fields[4])

		records = append(records, audit.AppRecord{
			Name:         name,
			PkgID:        name,
			Version:      version,
			Author:       publisher,
			Source:       audit.SourceSnap,
			ReinstallCmd: snapReinstall(name, channel),
		})
	}
	return records
}

// stripVerifiedMark removes the trailing ✓ that `snap list` appends
// to verified publishers when --unicode=never is unavailable.
func stripVerifiedMark(s string) string {
	s = strings.TrimSuffix(s, "✓")
	s = strings.TrimSuffix(s, "**")
	return strings.TrimSpace(s)
}

// snapReinstall picks the channel to pass back to `snap install`.
// If the package is on a stable channel we omit the flag; otherwise
// we record the explicit --channel.
func snapReinstall(name, channel string) string {
	channel = strings.TrimSpace(channel)
	if channel == "" || channel == "latest/stable" || channel == "stable" {
		return "snap install " + name
	}
	return "snap install " + name + " --channel=" + channel
}
