package macos

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"howett.net/plist"

	"github.com/simtabi/osaat/internal/audit"
)

// infoPlist captures the fields we read out of <App>.app/Contents/Info.plist.
// The plist struct tag matches howett.net/plist's decoder.
type infoPlist struct {
	CFBundleName               string `plist:"CFBundleName"`
	CFBundleDisplayName        string `plist:"CFBundleDisplayName"`
	CFBundleIdentifier         string `plist:"CFBundleIdentifier"`
	CFBundleShortVersionString string `plist:"CFBundleShortVersionString"`
	CFBundleVersion            string `plist:"CFBundleVersion"`
	CFBundleExecutable         string `plist:"CFBundleExecutable"`
	LSApplicationCategoryType  string `plist:"LSApplicationCategoryType"`
}

type discoverRoot struct {
	path          string
	defaultSource audit.Source
}

// discover walks the well-known macOS application roots and returns one
// AppRecord per .app bundle found. Records contain the fields readable
// from the filesystem and Info.plist; enrichers fill in the rest.
func (c *Collector) discover(ctx context.Context) ([]audit.AppRecord, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	roots := []discoverRoot{
		{"/Applications", audit.SourceUnknown},
		{"/Applications/Utilities", audit.SourceUnknown},
		{filepath.Join(home, "Applications"), audit.SourceUnknown},
		{"/System/Applications", audit.SourceSystem},
		{"/System/Applications/Utilities", audit.SourceSystem},
	}

	var records []audit.AppRecord
	seen := make(map[string]struct{})

	for _, root := range roots {
		if _, err := os.Stat(root.path); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		entries, err := os.ReadDir(root.path)
		if err != nil {
			c.log.Warn("read root", "root", root.path, "err", err)
			continue
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".app") {
				continue
			}
			appPath := filepath.Join(root.path, e.Name())
			if _, dup := seen[appPath]; dup {
				continue
			}
			seen[appPath] = struct{}{}

			rec, err := readApp(appPath, root.defaultSource)
			if err != nil {
				c.log.Warn("read app", "path", appPath, "err", err)
				continue
			}
			records = append(records, rec)
		}
	}

	return records, nil
}

// readApp builds an AppRecord from a single .app bundle.
func readApp(appPath string, defaultSource audit.Source) (audit.AppRecord, error) {
	info, err := readInfoPlist(filepath.Join(appPath, "Contents", "Info.plist"))
	if err != nil {
		return audit.AppRecord{}, err
	}

	name := info.CFBundleDisplayName
	if name == "" {
		name = info.CFBundleName
	}
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(appPath), ".app")
	}

	version := info.CFBundleShortVersionString
	if version == "" {
		version = info.CFBundleVersion
	}

	var installedAt *time.Time
	if stat, err := os.Stat(appPath); err == nil {
		t := stat.ModTime()
		installedAt = &t
	}

	size, _ := dirSize(appPath)

	if defaultSource == "" {
		defaultSource = audit.SourceUnknown
	}

	return audit.AppRecord{
		Name:        name,
		BundleID:    info.CFBundleIdentifier,
		Version:     version,
		Source:      defaultSource,
		Path:        appPath,
		SizeBytes:   size,
		InstalledAt: installedAt,
	}, nil
}

// readInfoPlist decodes the Info.plist into a permissive map first and
// then extracts the string fields we care about. This is robust to
// apps that ship a dict or array where a string was expected (which
// otherwise crashes the strict struct-tag decoder).
func readInfoPlist(path string) (infoPlist, error) {
	f, err := os.Open(path)
	if err != nil {
		return infoPlist{}, err
	}
	defer f.Close()

	var raw map[string]any
	if err := plist.NewDecoder(f).Decode(&raw); err != nil {
		return infoPlist{}, err
	}

	return infoPlist{
		CFBundleName:               asString(raw["CFBundleName"]),
		CFBundleDisplayName:        asString(raw["CFBundleDisplayName"]),
		CFBundleIdentifier:         asString(raw["CFBundleIdentifier"]),
		CFBundleShortVersionString: asString(raw["CFBundleShortVersionString"]),
		CFBundleVersion:            asString(raw["CFBundleVersion"]),
		CFBundleExecutable:         asString(raw["CFBundleExecutable"]),
		LSApplicationCategoryType:  asString(raw["LSApplicationCategoryType"]),
	}, nil
}

// asString coerces an arbitrary plist value into a string when sensible
// and returns the empty string otherwise. Dict / array / date values
// for fields we expected to be string-typed are silently dropped.
func asString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case nil:
		return ""
	default:
		return ""
	}
}

// dirSize returns the sum of file sizes under root. Errors on individual
// files are ignored — a partial total is more useful than zero.
func dirSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total, err
}

// recordsByPath builds a lookup keyed by absolute path. Enrichers that
// know paths use this to attach data to existing records.
func recordsByPath(records []audit.AppRecord) map[string]*audit.AppRecord {
	out := make(map[string]*audit.AppRecord, len(records))
	for i := range records {
		out[records[i].Path] = &records[i]
	}
	return out
}

// recordsByBundleID builds a lookup keyed by CFBundleIdentifier. Records
// with an empty BundleID are skipped.
func recordsByBundleID(records []audit.AppRecord) map[string]*audit.AppRecord {
	out := make(map[string]*audit.AppRecord, len(records))
	for i := range records {
		if records[i].BundleID == "" {
			continue
		}
		out[records[i].BundleID] = &records[i]
	}
	return out
}

// recordsByLowerName builds a lookup keyed by case-folded app name.
// Useful for cross-referencing tools that report by name (mas, brew cask).
func recordsByLowerName(records []audit.AppRecord) map[string]*audit.AppRecord {
	out := make(map[string]*audit.AppRecord, len(records))
	for i := range records {
		out[strings.ToLower(records[i].Name)] = &records[i]
	}
	return out
}
