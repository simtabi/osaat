package macos

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/collectors"
)

// enrichFromPkgutil runs `pkgutil --pkgs` once and marks apps whose
// CFBundleIdentifier exactly matches a known pkg ID with
// Source = pkg_installer. This is conservative: many .pkg installers
// use a different bundle id from the app they install, so most apps
// won't match here. A future pass can call `pkgutil --files <pkg>` for
// a more complete cross-reference, but that's expensive.
func (c *Collector) enrichFromPkgutil(ctx context.Context, records []audit.AppRecord) error {
	if !collectors.LookupExe("pkgutil") {
		return nil
	}
	out, err := c.runCmd(ctx, "pkgutil", "--pkgs")
	if err != nil {
		return fmt.Errorf("pkgutil --pkgs: %w", err)
	}
	pkgs := parsePkgList(out)
	if len(pkgs) == 0 {
		return nil
	}

	pkgSet := make(map[string]struct{}, len(pkgs))
	for _, p := range pkgs {
		pkgSet[p] = struct{}{}
	}

	byBundleID := recordsByBundleID(records)
	for id, rec := range byBundleID {
		if _, ok := pkgSet[id]; ok {
			// Don't overwrite a more specific source already set by
			// the App Store / Homebrew enrichers.
			if rec.Source == audit.SourceUnknown || rec.Source == audit.SourceDMG {
				rec.Source = audit.SourcePkg
			}
		}
	}
	return nil
}

func parsePkgList(out []byte) []string {
	var pkgs []string
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		pkgs = append(pkgs, line)
	}
	return pkgs
}
