package macos

import (
	"context"
	"strings"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/collectors"
)

// enrichFromCodesign runs `codesign -dv --verbose=4 <path>` per app and
// fills in SigningStatus, SigningTeam, and Author (if not already set).
//
// codesign writes its primary output to stderr and may exit non-zero
// even when the output is useful, so we go through RunCmdCombined which
// captures both streams and never returns an error.
func (c *Collector) enrichFromCodesign(ctx context.Context, records []audit.AppRecord) error {
	if !collectors.LookupExe("codesign") {
		return nil
	}
	c.runPerApp(ctx, records, func(ctx context.Context, i int) {
		if records[i].Path == "" {
			return
		}
		out := collectors.RunCmdCombined(ctx, "codesign", "-dv", "--verbose=4", records[i].Path)
		status, team, authority := parseCodesign(string(out))
		records[i].SigningStatus = status
		if team != "" {
			records[i].SigningTeam = team
		}
		if authority != "" && records[i].Author == "" {
			records[i].Author = authority
		}
		if status == audit.SigningUnsigned {
			records[i].Note("unsigned binary")
		}
	})
	return nil
}

// parseCodesign extracts (status, team, primary authority) from the
// combined stderr+stdout of `codesign -dv --verbose=4`.
func parseCodesign(out string) (audit.SigningStatus, string, string) {
	if strings.Contains(out, "not signed at all") || strings.Contains(out, "code object is not signed") {
		return audit.SigningUnsigned, "", ""
	}

	var team, authority string
	status := audit.SigningUnknown

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "TeamIdentifier="):
			team = strings.TrimPrefix(line, "TeamIdentifier=")
		case strings.HasPrefix(line, "Authority=") && authority == "":
			authority = strings.TrimPrefix(line, "Authority=")
			status = audit.SigningSigned
		case strings.HasPrefix(line, "Signature="):
			val := strings.TrimPrefix(line, "Signature=")
			if val == "adhoc" {
				status = audit.SigningAdHoc
			} else if val != "" && status == audit.SigningUnknown {
				status = audit.SigningSigned
			}
		}
	}

	if team == "not set" {
		team = ""
	}
	return status, team, authority
}

var _ collectors.Collector = (*Collector)(nil)
