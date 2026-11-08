package macos

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/collectors"
)

// brewCaskEntry is one row from `brew list --cask --versions`.
type brewCaskEntry struct {
	Cask    string
	Version string
}

// enrichFromBrew runs `brew list --cask --versions` once and matches
// casks against discovered apps by case-folded name (with hyphens
// turned into spaces to match the typical cask-vs-app-name pattern).
//
// Apps matched here get Source = homebrew_cask and a `brew install
// --cask <name>` reinstall command. Skipped if brew is not on PATH.
//
// Homebrew formulae are not cross-referenced — formulae rarely
// correspond to /Applications bundles. They may surface in a future
// "command-line tools" collector.
func (c *Collector) enrichFromBrew(ctx context.Context, records []audit.AppRecord) error {
	if !collectors.LookupExe("brew") {
		return nil
	}
	out, err := c.runCmd(ctx, "brew", "list", "--cask", "--versions")
	if err != nil {
		return fmt.Errorf("brew list --cask: %w", err)
	}
	entries := parseBrewCaskList(out)
	if len(entries) == 0 {
		return nil
	}

	byName := recordsByLowerName(records)
	for _, e := range entries {
		// brew cask names are kebab-case ("microsoft-teams") while the
		// discovered app names are space-separated ("Microsoft Teams").
		// Try both forms.
		candidates := []string{
			strings.ToLower(e.Cask),
			strings.ReplaceAll(strings.ToLower(e.Cask), "-", " "),
		}
		var rec *audit.AppRecord
		for _, cand := range candidates {
			if r, ok := byName[cand]; ok {
				rec = r
				break
			}
		}
		if rec == nil {
			continue
		}
		rec.Source = audit.SourceBrewCask
		rec.ReinstallCmd = "brew install --cask " + e.Cask
	}
	return nil
}

func parseBrewCaskList(out []byte) []brewCaskEntry {
	var entries []brewCaskEntry
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		switch len(fields) {
		case 0:
			continue
		case 1:
			entries = append(entries, brewCaskEntry{Cask: fields[0]})
		default:
			entries = append(entries, brewCaskEntry{Cask: fields[0], Version: fields[1]})
		}
	}
	return entries
}
