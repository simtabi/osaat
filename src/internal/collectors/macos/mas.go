package macos

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/collectors"
)

// masLineRe matches a `mas list` line:
//
//	1234567890 App Name    (1.2.3)
//
// Captures: id, name, version.
var masLineRe = regexp.MustCompile(`^(\d+)\s+(.+?)\s+\(([^)]+)\)\s*$`)

// masEntry is one row from `mas list`.
type masEntry struct {
	ID      string
	Name    string
	Version string
}

// enrichFromMas runs `mas list` once and cross-references its output by
// app name. Apps matched here have their Source set to App Store,
// AppStoreID populated, and a `mas install <id>` reinstall command.
// Skipped if mas is not on PATH (mas is optional; not every Mac has it).
func (c *Collector) enrichFromMas(ctx context.Context, records []audit.AppRecord) error {
	if !collectors.LookupExe("mas") {
		return nil
	}
	out, err := c.runCmd(ctx, "mas", "list")
	if err != nil {
		return fmt.Errorf("mas list: %w", err)
	}
	entries := parseMasList(out)
	if len(entries) == 0 {
		return nil
	}

	byName := recordsByLowerName(records)
	for _, e := range entries {
		rec := byName[strings.ToLower(e.Name)]
		if rec == nil {
			continue
		}
		rec.Source = audit.SourceAppStore
		rec.AppStoreID = e.ID
		rec.ReinstallCmd = "mas install " + e.ID
		if rec.Version == "" {
			rec.Version = e.Version
		}
	}
	return nil
}

func parseMasList(out []byte) []masEntry {
	var entries []masEntry
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		m := masLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		entries = append(entries, masEntry{ID: m[1], Name: m[2], Version: m[3]})
	}
	return entries
}
