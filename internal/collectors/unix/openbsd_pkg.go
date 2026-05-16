package unix

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/simtabi/osaat/internal/audit"
)

// pkgInfoLine matches a line of `pkg_info` output:
//
//	<name>-<version>[-<flavor>]  <one-line description>
//
// We capture name and version+flavor as a single string. Some
// packages (especially NetBSD) include a trailing "nb<n>" suffix on
// the version; that's part of the version, not the name.
var pkgInfoLine = regexp.MustCompile(`^(\S+?)-([0-9][^\s]*)\s+(.+)$`)

// collectPkgInfo runs `pkg_info` (OpenBSD/NetBSD/DragonflyBSD) and
// parses each line. The output format has been stable since the
// 1990s, but it's whitespace-formatted rather than tab-delimited,
// so we use a regex.
func (c *Collector) collectPkgInfo(ctx context.Context) ([]audit.AppRecord, error) {
	out, err := c.runCmd(ctx, "pkg_info")
	if err != nil {
		return nil, fmt.Errorf("pkg_info: %w", err)
	}
	return parsePkgInfoOutput(out), nil
}

// parsePkgInfoOutput is the pure parser.
func parsePkgInfoOutput(out []byte) []audit.AppRecord {
	var records []audit.AppRecord
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), " \t")
		if line == "" {
			continue
		}
		m := pkgInfoLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		version := m[2]
		// description (m[3]) is currently dropped — we don't have a
		// dedicated Description field on AppRecord, and the
		// CollectorNotes slot is reserved for soft warnings.

		records = append(records, audit.AppRecord{
			Name:         name,
			PkgID:        name,
			Version:      version,
			Source:       audit.SourceBSDPkg,
			ReinstallCmd: "pkg_add " + name,
		})
	}
	return records
}
