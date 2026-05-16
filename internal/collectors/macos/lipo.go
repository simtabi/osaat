package macos

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/collectors"
)

// enrichFromLipo runs `lipo -archs <executable>` for every record and
// sets AppleSilicon to true if the binary contains an arm64 slice,
// false if it's Intel-only, and leaves it nil when the executable
// can't be located or the lipo invocation fails.
//
// The executable is at <App>.app/Contents/MacOS/<CFBundleExecutable>.
// We re-read Info.plist here rather than caching it on the AppRecord
// to keep the record schema clean of macOS-internal fields.
func (c *Collector) enrichFromLipo(ctx context.Context, records []audit.AppRecord) error {
	if !collectors.LookupExe("lipo") {
		return nil
	}
	c.runPerApp(ctx, records, func(ctx context.Context, i int) {
		exe := executableFor(records[i])
		if exe == "" {
			return
		}
		out, err := c.runCmd(ctx, "lipo", "-archs", exe)
		if err != nil {
			return
		}
		archs := strings.Fields(strings.TrimSpace(string(out)))
		if len(archs) == 0 {
			return
		}
		hasARM64 := false
		for _, a := range archs {
			if a == "arm64" || a == "arm64e" {
				hasARM64 = true
				break
			}
		}
		records[i].AppleSilicon = boolPtr(hasARM64)
		if !hasARM64 {
			records[i].Note("intel-only binary (Rosetta required on Apple Silicon)")
		}
	})
	return nil
}

func executableFor(r audit.AppRecord) string {
	if r.Path == "" {
		return ""
	}
	info, err := readInfoPlist(filepath.Join(r.Path, "Contents", "Info.plist"))
	if err != nil || info.CFBundleExecutable == "" {
		return ""
	}
	return filepath.Join(r.Path, "Contents", "MacOS", info.CFBundleExecutable)
}

func boolPtr(b bool) *bool { return &b }
