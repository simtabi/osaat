package macos

import (
	"context"
	"fmt"

	"howett.net/plist"

	"github.com/simtabi/osaat/internal/audit"
)

// sysProfilerApp captures the fields we read out of
// `system_profiler -xml SPApplicationsDataType`. Field names match the
// keys macOS uses in the plist output. We deliberately omit
// `lastModified` (date-typed) and `info` (sometimes dict-typed) so the
// decoder doesn't reject the whole document on a type mismatch.
type sysProfilerApp struct {
	Name         string   `plist:"_name"`
	Path         string   `plist:"path"`
	Version      string   `plist:"version"`
	ObtainedFrom string   `plist:"obtained_from"`
	SignedBy     []string `plist:"signed_by"`
}

// sysProfilerSection is the outer wrapper in the SPApplicationsDataType
// output — one section with all applications under "_items".
type sysProfilerSection struct {
	Items []sysProfilerApp `plist:"_items"`
}

// enrichFromSystemProfiler runs `system_profiler -xml
// SPApplicationsDataType`, parses it, and refines Source / Author /
// Version on records whose path matches.
func (c *Collector) enrichFromSystemProfiler(ctx context.Context, records []audit.AppRecord) error {
	out, err := c.runCmd(ctx, "system_profiler", "-xml", "SPApplicationsDataType")
	if err != nil {
		return fmt.Errorf("system_profiler: %w", err)
	}

	var sections []sysProfilerSection
	if _, err := plist.Unmarshal(out, &sections); err != nil {
		return fmt.Errorf("parse system_profiler xml: %w", err)
	}

	byPath := recordsByPath(records)
	for _, sec := range sections {
		for _, app := range sec.Items {
			rec := byPath[app.Path]
			if rec == nil {
				continue
			}
			rec.Source = mapObtainedFrom(app.ObtainedFrom, rec.Source)
			if rec.Version == "" {
				rec.Version = app.Version
			}
			if rec.Author == "" && len(app.SignedBy) > 0 {
				rec.Author = app.SignedBy[0]
			}
		}
	}
	return nil
}

// mapObtainedFrom converts the system_profiler "obtained_from" string
// into a stable Source value, preserving the existing Source on a record
// when system_profiler reports "unknown" or has no opinion.
func mapObtainedFrom(obtained string, current audit.Source) audit.Source {
	switch obtained {
	case "mac_app_store":
		return audit.SourceAppStore
	case "identified_developer":
		// A signed developer ID — could be brew cask, direct DMG, or
		// pkg installer. Leave it for later enrichers to refine; mark
		// as DMG-or-direct only if nothing else has claimed it yet.
		if current == audit.SourceUnknown || current == "" {
			return audit.SourceDMG
		}
		return current
	case "apple":
		return audit.SourceSystem
	case "unknown":
		return current
	default:
		return current
	}
}

