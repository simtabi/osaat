package linux

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/simtabi/osaat/internal/audit"
)

// collectPacman runs `pacman -Q` (name + version per line) and
// produces AppRecords. We deliberately don't follow up with
// `pacman -Qi <name>` for richer metadata — that would mean N
// subprocess calls on a system that can easily have 1500 packages,
// trading scan time for fields most users won't read. Phase 4 can
// revisit if a use case emerges.
func (c *Collector) collectPacman(ctx context.Context) ([]audit.AppRecord, error) {
	out, err := c.runCmd(ctx, "pacman", "-Q")
	if err != nil {
		return nil, fmt.Errorf("pacman -Q: %w", err)
	}
	return parsePacmanOutput(out), nil
}

// parsePacmanOutput parses `pacman -Q` output: one "name version"
// per line, separated by a single space.
func parsePacmanOutput(out []byte) []audit.AppRecord {
	var records []audit.AppRecord
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		// Use Fields so multiple spaces collapse, but keep the rest
		// of the line as a single version string — Arch versions can
		// include hyphens and pkgrel suffixes but never spaces.
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		version := strings.Join(parts[1:], " ")
		records = append(records, audit.AppRecord{
			Name:         name,
			PkgID:        name,
			Version:      version,
			Source:       audit.SourcePacman,
			ReinstallCmd: "pacman -S " + name,
		})
	}
	return records
}
