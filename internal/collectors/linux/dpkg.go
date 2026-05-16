package linux

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/simtabi/osaat/internal/audit"
)

// dpkgQueryArgs is the tab-separated format we request from
// `dpkg-query -W`. The trailing \n separates rows.
var dpkgQueryArgs = []string{
	"-W",
	"-f=${Package}\t${Version}\t${Maintainer}\t${Installed-Size}\t${db:Status-Status}\t${Section}\t${Homepage}\n",
}

// collectDpkg runs `dpkg-query -W ...` once and parses each line into
// an AppRecord. Packages whose status is not "installed" are skipped
// — dpkg tracks half-installed / config-only packages we don't want
// in the inventory.
func (c *Collector) collectDpkg(ctx context.Context) ([]audit.AppRecord, error) {
	out, err := c.runCmd(ctx, "dpkg-query", dpkgQueryArgs...)
	if err != nil {
		return nil, fmt.Errorf("dpkg-query: %w", err)
	}
	return parseDpkgOutput(out), nil
}

// parseDpkgOutput is the pure parser separated from the subprocess
// boundary so tests can hand it captured fixture bytes.
func parseDpkgOutput(out []byte) []audit.AppRecord {
	var records []audit.AppRecord
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Split(line, "\t")
		if len(fields) < 5 {
			continue
		}
		status := strings.TrimSpace(fields[4])
		if status != "installed" {
			continue
		}
		name := strings.TrimSpace(fields[0])
		if name == "" {
			continue
		}
		author, _ := extractEmailFromMaintainer(fields[2])

		var size int64
		if v := strings.TrimSpace(fields[3]); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				// Installed-Size is reported in kilobytes by dpkg.
				size = n * 1024
			}
		}

		var homepage string
		if len(fields) > 6 {
			homepage = strings.TrimSpace(fields[6])
		}

		records = append(records, audit.AppRecord{
			Name:         name,
			PkgID:        name,
			Version:      strings.TrimSpace(fields[1]),
			Author:       author,
			VendorURL:    homepage,
			Source:       audit.SourceDpkg,
			SizeBytes:    size,
			ReinstallCmd: "apt install " + name,
		})
	}
	return records
}

// epochToTime is a small helper used by other Linux parsers that
// emit epoch seconds in their format strings. Returns nil on parse
// failure or zero values so downstream JSON omits the field.
func epochToTime(s string) *time.Time {
	v := strings.TrimSpace(s)
	if v == "" || v == "0" {
		return nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	t := time.Unix(n, 0).UTC()
	return &t
}
