package unix

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

// freebsdPkgArgs asks FreeBSD pkg to emit a tab-separated row per
// package: name, version, maintainer, install timestamp (epoch),
// origin (category/name), www URL.
var freebsdPkgArgs = []string{
	"query",
	"-a",
	"%n\t%v\t%m\t%t\t%o\t%w\n",
}

// collectFreeBSDPkg runs `pkg query -a ...` and parses every row.
func (c *Collector) collectFreeBSDPkg(ctx context.Context) ([]audit.AppRecord, error) {
	out, err := c.runCmd(ctx, "pkg", freebsdPkgArgs...)
	if err != nil {
		return nil, fmt.Errorf("pkg query: %w", err)
	}
	return parseFreeBSDPkgOutput(out), nil
}

// parseFreeBSDPkgOutput is the pure parser.
func parseFreeBSDPkgOutput(out []byte) []audit.AppRecord {
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
		version := strings.TrimSpace(fields[1])
		maintainer := strings.TrimSpace(fields[2])

		var url string
		if len(fields) >= 6 {
			url = strings.TrimSpace(fields[5])
		}

		rec := audit.AppRecord{
			Name:         name,
			PkgID:        name,
			Version:      version,
			Author:       maintainer,
			VendorURL:    url,
			Source:       audit.SourceBSDPkg,
			ReinstallCmd: "pkg install -y " + name,
		}
		if len(fields) >= 4 {
			rec.InstalledAt = epochSecondsToTimePtr(fields[3])
		}
		records = append(records, rec)
	}
	return records
}

// epochSecondsToTimePtr parses an epoch-seconds string and returns
// a *time.Time, or nil on empty / zero / invalid input.
func epochSecondsToTimePtr(s string) *time.Time {
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
