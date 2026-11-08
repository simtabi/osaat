package macos

import (
	"context"
	"regexp"
	"strings"

	"github.com/simtabi/osaat/internal/audit"
)

var mdlsURLRe = regexp.MustCompile(`"([^"]+)"`)

// enrichFromMdls populates DownloadURL from the kMDItemWhereFroms
// Spotlight attribute. The attribute is a tuple of strings; the first
// element is typically the direct download URL and the second is the
// referrer page. We keep the first.
//
// Missing attributes are not an error — many apps (system, App Store,
// or installed by a package manager) never had a quarantine-source URL.
// Runs in parallel across records via runPerApp.
func (c *Collector) enrichFromMdls(ctx context.Context, records []audit.AppRecord) error {
	c.runPerApp(ctx, records, func(ctx context.Context, i int) {
		if records[i].Path == "" || records[i].DownloadURL != "" {
			return
		}
		out, err := c.runCmd(ctx, "mdls", "-name", "kMDItemWhereFroms", records[i].Path)
		if err != nil {
			return
		}
		if url := parseMdlsWhereFroms(out); url != "" {
			records[i].DownloadURL = url
		}
	})
	return nil
}

// parseMdlsWhereFroms extracts the first URL from mdls output of the
// form:
//
//	kMDItemWhereFroms = (
//	    "https://example.com/foo.dmg",
//	    "https://example.com/"
//	)
//
// or returns "" when the attribute is absent or "(null)".
func parseMdlsWhereFroms(out []byte) string {
	s := string(out)
	if strings.Contains(s, "(null)") {
		return ""
	}
	matches := mdlsURLRe.FindAllStringSubmatch(s, -1)
	for _, m := range matches {
		v := strings.TrimSpace(m[1])
		if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
			return v
		}
	}
	return ""
}

