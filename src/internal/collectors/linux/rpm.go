package linux

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/simtabi/osaat/internal/audit"
)

// rpmQueryArgs requests a tab-separated row per package.
// %{INSTALLTIME} is epoch seconds; %{SIZE} is bytes (not KB).
var rpmQueryArgs = []string{
	"-qa",
	"--qf=%{NAME}\t%{VERSION}\t%{VENDOR}\t%{SIZE}\t%{INSTALLTIME}\t%{URL}\n",
}

// collectRpm queries every installed RPM and produces AppRecords.
// The reinstall command is `dnf install <name>` on dnf-based distros
// (Fedora, RHEL 8+, CentOS Stream); on legacy yum systems
// `yum install <name>` works identically. We pick one — dnf — and
// document the substitution in docs/tools/scan.md.
func (c *Collector) collectRpm(ctx context.Context) ([]audit.AppRecord, error) {
	out, err := c.runCmd(ctx, "rpm", rpmQueryArgs...)
	if err != nil {
		return nil, fmt.Errorf("rpm -qa: %w", err)
	}
	return parseRpmOutput(out, detectRpmFront()), nil
}

// parseRpmOutput is the pure parser.
func parseRpmOutput(out []byte, frontend string) []audit.AppRecord {
	if frontend == "" {
		frontend = "dnf"
	}
	var records []audit.AppRecord
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		fields := strings.Split(sc.Text(), "\t")
		if len(fields) < 4 {
			continue
		}
		name := strings.TrimSpace(fields[0])
		if name == "" {
			continue
		}
		var size int64
		if n, err := strconv.ParseInt(strings.TrimSpace(fields[3]), 10, 64); err == nil {
			size = n
		}
		var url string
		if len(fields) >= 6 {
			url = strings.TrimSpace(fields[5])
		}

		rec := audit.AppRecord{
			Name:         name,
			PkgID:        name,
			Version:      strings.TrimSpace(fields[1]),
			Author:       strings.TrimSpace(fields[2]),
			VendorURL:    url,
			Source:       audit.SourceRpm,
			SizeBytes:    size,
			ReinstallCmd: frontend + " install " + name,
		}
		if len(fields) >= 5 {
			if t := epochToTime(fields[4]); t != nil {
				rec.InstalledAt = t
			}
		}
		// rpm emits "(none)" for missing fields — clean these up.
		rec.Author = cleanRPMNone(rec.Author)
		rec.VendorURL = cleanRPMNone(rec.VendorURL)
		records = append(records, rec)
	}
	return records
}

// cleanRPMNone replaces RPM's "(none)" placeholder with an empty
// string so the JSON output stays terse.
func cleanRPMNone(s string) string {
	if s == "(none)" {
		return ""
	}
	return s
}

// detectRpmFront returns the rpm-frontend likely to be installed —
// dnf on Fedora/RHEL 8+, yum on older systems. Falls back to dnf
// since the rpm package itself is necessary to even query, and dnf
// is the modern default.
func detectRpmFront() string {
	if _, err := os.Stat("/etc/os-release"); err == nil {
		if data, err := os.ReadFile("/etc/os-release"); err == nil {
			lc := strings.ToLower(string(data))
			switch {
			case strings.Contains(lc, `id=fedora`),
				strings.Contains(lc, `id=rhel`),
				strings.Contains(lc, `id="rhel"`):
				return "dnf"
			case strings.Contains(lc, `id=opensuse`),
				strings.Contains(lc, `id="opensuse"`),
				strings.Contains(lc, `id=sles`):
				return "zypper"
			}
		}
	}
	return "dnf"
}
