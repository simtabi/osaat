package linux

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/simtabi/osaat/internal/audit"
)

// appImageRoots is the set of locations osaat scans for *.AppImage
// files. Walks are non-recursive — bundles drop a single executable
// at the top level of these directories on the vast majority of
// systems.
var appImageRoots = []string{
	"~/.local/bin",
	"~/Applications",
	"~/AppImages",
	"~/Downloads",
}

// collectAppImage walks the well-known AppImage roots and produces
// one AppRecord per file matching *.AppImage. AppImages don't carry
// embedded version metadata reliably, so we record the file name,
// size, and install date and leave the rest blank.
func (c *Collector) collectAppImage(_ context.Context) ([]audit.AppRecord, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	var records []audit.AppRecord
	for _, root := range appImageRoots {
		path := strings.Replace(root, "~", home, 1)
		recs, err := scanAppImageDir(path)
		if err != nil {
			c.log.Warn("appimage scan", "root", path, "err", err)
			continue
		}
		records = append(records, recs...)
	}
	return records, nil
}

// scanAppImageDir lists the top level of dir and emits an AppRecord
// for every regular file whose name ends in .AppImage (case-insensitive).
func scanAppImageDir(dir string) ([]audit.AppRecord, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var records []audit.AppRecord
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Ext(e.Name()), ".AppImage") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		info, err := e.Info()
		if err != nil {
			continue
		}
		rec := buildAppImageRecord(path, e.Name(), info)
		records = append(records, rec)
	}
	return records, nil
}

// buildAppImageRecord produces the record for one AppImage file.
// The display name is the file name without the trailing ".AppImage";
// the path is recorded as-is. SizeBytes comes from the file stat.
func buildAppImageRecord(path, fileName string, info fs.FileInfo) audit.AppRecord {
	name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	installedAt := info.ModTime().UTC()
	return audit.AppRecord{
		Name:         name,
		Path:         path,
		Source:       audit.SourceAppImage,
		SizeBytes:    info.Size(),
		InstalledAt:  &installedAt,
		ReinstallCmd: "(redownload AppImage — vendor-specific)",
	}
}
