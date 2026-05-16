package linux

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/simtabi/osaat/internal/audit"
)

// flatpakColumns is the column set we ask flatpak to emit. The order
// here is the order parseFlatpakOutput expects.
var flatpakColumns = []string{
	"application",
	"name",
	"version",
	"branch",
	"origin",
	"installation",
}

// collectFlatpak runs `flatpak list --app --columns=...`. The
// --columns flag produces stable tab-separated output that's far
// easier to parse than the human-friendly default.
func (c *Collector) collectFlatpak(ctx context.Context) ([]audit.AppRecord, error) {
	out, err := c.runCmd(ctx, "flatpak", "list", "--app",
		"--columns="+strings.Join(flatpakColumns, ","))
	if err != nil {
		return nil, fmt.Errorf("flatpak list: %w", err)
	}
	return parseFlatpakOutput(out), nil
}

// parseFlatpakOutput parses tab-separated rows from
// `flatpak list --columns=application,name,version,branch,origin,installation`.
// Empty cells appear as "-" in flatpak's output.
func parseFlatpakOutput(out []byte) []audit.AppRecord {
	var records []audit.AppRecord
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		// Flatpak older than 1.0 prints a header row; modern versions
		// don't with --columns. Skip a row that doesn't have an ID-shaped
		// first column (lowercased, contains a dot).
		fields := strings.Split(line, "\t")
		if len(fields) < 6 {
			continue
		}
		appID := strings.TrimSpace(fields[0])
		if appID == "" || !strings.Contains(appID, ".") {
			continue
		}
		name := orDash(fields[1])
		if name == "" {
			name = appID
		}
		version := orDash(fields[2])
		branch := orDash(fields[3])
		origin := orDash(fields[4])

		records = append(records, audit.AppRecord{
			Name:         name,
			PkgID:        appID,
			Version:      version,
			Source:       audit.SourceFlatpak,
			ReinstallCmd: flatpakReinstall(appID, branch, origin),
		})
	}
	return records
}

// orDash converts flatpak's empty placeholder "-" into an empty string.
func orDash(s string) string {
	s = strings.TrimSpace(s)
	if s == "-" {
		return ""
	}
	return s
}

// flatpakReinstall builds the install command. When origin is known
// (typically "flathub") we include it so the new machine pulls from
// the same remote.
func flatpakReinstall(appID, branch, origin string) string {
	var b strings.Builder
	b.WriteString("flatpak install -y")
	if origin != "" {
		b.WriteString(" " + origin)
	}
	b.WriteString(" " + appID)
	if branch != "" && branch != "stable" {
		b.WriteString("//" + branch)
	}
	return b.String()
}
