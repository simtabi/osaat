package reporters

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/version"
)

// TextReporter emits a plain-text inventory: one record per block with
// labeled fields. Easy to grep, easy to read in a terminal, no
// rendering dependencies.
type TextReporter struct {
	Now func() time.Time
}

// NewTextReporter returns a text reporter that uses the real wall clock.
func NewTextReporter() *TextReporter {
	return &TextReporter{Now: func() time.Time { return time.Now().UTC() }}
}

// Format implements Reporter.
func (r *TextReporter) Format() string { return "txt" }

// Write implements Reporter.
func (r *TextReporter) Write(records []audit.AppRecord, w io.Writer) error {
	sorted := make([]audit.AppRecord, len(records))
	copy(sorted, records)
	sort.SliceStable(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	hostname, _ := os.Hostname()
	now := r.Now()

	var b strings.Builder
	fmt.Fprintln(&b, "osaat application inventory")
	fmt.Fprintln(&b, strings.Repeat("=", 60))
	fmt.Fprintf(&b, "Generated:    %s\n", now.Format(time.RFC3339))
	fmt.Fprintf(&b, "Host:         %s (%s/%s)\n", hostname, runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&b, "Tool:         osaat %s\n", version.Version)
	fmt.Fprintf(&b, "Applications: %d\n\n", len(sorted))

	bySource := map[audit.Source]int{}
	bySigning := map[audit.SigningStatus]int{}
	for _, rec := range sorted {
		bySource[rec.Source]++
		if rec.SigningStatus != "" {
			bySigning[rec.SigningStatus]++
		}
	}

	fmt.Fprintln(&b, "By source")
	fmt.Fprintln(&b, strings.Repeat("-", 40))
	for _, src := range orderedSources() {
		if n, ok := bySource[src]; ok && n > 0 {
			fmt.Fprintf(&b, "  %-22s %d\n", src, n)
		}
	}
	fmt.Fprintln(&b)

	if len(bySigning) > 0 {
		fmt.Fprintln(&b, "By signing status")
		fmt.Fprintln(&b, strings.Repeat("-", 40))
		for _, st := range []audit.SigningStatus{audit.SigningSigned, audit.SigningAdHoc, audit.SigningUnsigned, audit.SigningUnknown} {
			if n, ok := bySigning[st]; ok && n > 0 {
				fmt.Fprintf(&b, "  %-22s %d\n", st, n)
			}
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprintln(&b, "Applications")
	fmt.Fprintln(&b, strings.Repeat("-", 40))
	for _, rec := range sorted {
		fmt.Fprintf(&b, "%s", rec.Name)
		if rec.Version != "" {
			fmt.Fprintf(&b, " (%s)", rec.Version)
		}
		fmt.Fprintln(&b)
		writeIfSet(&b, "source",    string(rec.Source))
		writeIfSet(&b, "bundle",    rec.BundleID)
		writeIfSet(&b, "version",   rec.Version)
		writeIfSet(&b, "signing",   string(rec.SigningStatus))
		writeIfSet(&b, "team",      rec.SigningTeam)
		writeIfSet(&b, "author",    rec.Author)
		writeIfSet(&b, "path",      rec.Path)
		if rec.SizeBytes > 0 {
			writeIfSet(&b, "size", humanSize(rec.SizeBytes))
		}
		if rec.InstalledAt != nil {
			writeIfSet(&b, "installed", rec.InstalledAt.UTC().Format(time.RFC3339))
		}
		if rec.LastUsedAt != nil {
			writeIfSet(&b, "lastused", rec.LastUsedAt.UTC().Format(time.RFC3339))
		}
		writeIfSet(&b, "download",  rec.DownloadURL)
		writeIfSet(&b, "vendor",    rec.VendorURL)
		writeIfSet(&b, "reinstall", rec.ReinstallCmd)
		if len(rec.CollectorNotes) > 0 {
			writeIfSet(&b, "notes", strings.Join(rec.CollectorNotes, "; "))
		}
		fmt.Fprintln(&b)
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func writeIfSet(b *strings.Builder, label, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(b, "    %-10s %s\n", label+":", value)
}

func orderedSources() []audit.Source {
	return []audit.Source{
		audit.SourceAppStore,
		audit.SourceBrewCask,
		audit.SourceBrewFormula,
		audit.SourcePkg,
		audit.SourceDMG,
		audit.SourceSystem,
		audit.SourceSandbox,
		audit.SourceDpkg,
		audit.SourceRpm,
		audit.SourcePacman,
		audit.SourceSnap,
		audit.SourceFlatpak,
		audit.SourceAppImage,
		audit.SourceBSDPkg,
		audit.SourceUnknown,
	}
}
