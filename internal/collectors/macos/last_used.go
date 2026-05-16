package macos

import (
	"context"
	"strings"
	"time"

	"github.com/simtabi/osaat/internal/audit"
)

// mdlsDateLayouts holds the formats `mdls -raw` emits for date
// attributes. The exact format varies by macOS version and locale —
// we try each in order.
var mdlsDateLayouts = []string{
	"2006-01-02 15:04:05 -0700",
	"2006-01-02 15:04:05 +0000",
	"2006-01-02 15:04:05 -0700 MST",
	time.RFC3339,
}

// enrichFromLastUsed populates AppRecord.LastUsedAt from Spotlight's
// kMDItemLastUsedDate. Apps that have never been opened return
// "(null)" and are left zero-valued — the reporter / forgotten-apps
// logic interprets a zero time as "no signal".
func (c *Collector) enrichFromLastUsed(ctx context.Context, records []audit.AppRecord) error {
	c.runPerApp(ctx, records, func(ctx context.Context, i int) {
		if records[i].Path == "" {
			return
		}
		out, err := c.runCmd(ctx, "mdls", "-name", "kMDItemLastUsedDate", "-raw", records[i].Path)
		if err != nil {
			return
		}
		t := parseMdlsDate(string(out))
		if !t.IsZero() {
			records[i].LastUsedAt = &t
		}
	})
	return nil
}

// parseMdlsDate returns the parsed time, or the zero time when the
// value is "(null)", empty, or in an unrecognized format.
func parseMdlsDate(raw string) time.Time {
	s := strings.TrimSpace(raw)
	if s == "" || s == "(null)" {
		return time.Time{}
	}
	for _, layout := range mdlsDateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
