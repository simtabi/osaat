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

// MarkdownReporter writes a GitHub-flavored Markdown report with a
// summary, per-source breakdowns, and the main application table.
type MarkdownReporter struct {
	Now func() time.Time
}

// NewMarkdownReporter returns a reporter that uses the real wall clock.
func NewMarkdownReporter() *MarkdownReporter {
	return &MarkdownReporter{Now: func() time.Time { return time.Now().UTC() }}
}

// Format implements Reporter.
func (r *MarkdownReporter) Format() string { return "markdown" }

// Write implements Reporter.
func (r *MarkdownReporter) Write(records []audit.AppRecord, w io.Writer) error {
	sorted := make([]audit.AppRecord, len(records))
	copy(sorted, records)
	sort.SliceStable(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	hostname, _ := os.Hostname()
	now := r.Now()

	var b strings.Builder
	fmt.Fprintf(&b, "# Application inventory\n\n")
	fmt.Fprintf(&b, "Generated: %s\n\n", now.Format(time.RFC3339))
	fmt.Fprintf(&b, "| | |\n|---|---|\n")
	fmt.Fprintf(&b, "| Host | `%s` |\n", hostname)
	fmt.Fprintf(&b, "| OS | `%s/%s` |\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&b, "| Tool | `osaat %s` |\n", version.Version)
	fmt.Fprintf(&b, "| Applications | %d |\n\n", len(sorted))

	writeSummary(&b, sorted)
	writeAppTable(&b, sorted)

	if notes := collectNotedApps(sorted); len(notes) > 0 {
		fmt.Fprintf(&b, "## Apps with collector notes\n\n")
		fmt.Fprintf(&b, "| Name | Notes |\n|---|---|\n")
		for _, rec := range notes {
			fmt.Fprintf(&b, "| %s | %s |\n", escapeMD(rec.Name), escapeMD(strings.Join(rec.CollectorNotes, "; ")))
		}
		fmt.Fprintln(&b)
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func writeSummary(b *strings.Builder, records []audit.AppRecord) {
	bySource := make(map[audit.Source]int)
	bySigning := make(map[audit.SigningStatus]int)
	for _, r := range records {
		bySource[r.Source]++
		if r.SigningStatus != "" {
			bySigning[r.SigningStatus]++
		}
	}

	fmt.Fprintf(b, "## Summary\n\n")
	fmt.Fprintf(b, "### By source\n\n| Source | Count |\n|---|---|\n")
	for _, src := range []audit.Source{
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
	} {
		if n, ok := bySource[src]; ok && n > 0 {
			fmt.Fprintf(b, "| `%s` | %d |\n", src, n)
		}
	}
	fmt.Fprintln(b)

	if len(bySigning) > 0 {
		fmt.Fprintf(b, "### By signing status\n\n| Status | Count |\n|---|---|\n")
		for _, st := range []audit.SigningStatus{
			audit.SigningSigned,
			audit.SigningAdHoc,
			audit.SigningUnsigned,
			audit.SigningUnknown,
		} {
			if n, ok := bySigning[st]; ok && n > 0 {
				fmt.Fprintf(b, "| `%s` | %d |\n", st, n)
			}
		}
		fmt.Fprintln(b)
	}
}

func writeAppTable(b *strings.Builder, records []audit.AppRecord) {
	fmt.Fprintf(b, "## Applications\n\n")
	fmt.Fprintf(b, "| Name | Version | Source | Signing | Author | Size | Reinstall |\n")
	fmt.Fprintf(b, "|---|---|---|---|---|---|---|\n")
	for _, r := range records {
		fmt.Fprintf(b, "| %s | %s | `%s` | `%s` | %s | %s | %s |\n",
			escapeMD(r.Name),
			escapeMD(r.Version),
			r.Source,
			r.SigningStatus,
			escapeMD(r.Author),
			humanSize(r.SizeBytes),
			escapeMD(r.ReinstallCmd),
		)
	}
	fmt.Fprintln(b)
}

func collectNotedApps(records []audit.AppRecord) []audit.AppRecord {
	var out []audit.AppRecord
	for _, r := range records {
		if len(r.CollectorNotes) > 0 {
			out = append(out, r)
		}
	}
	return out
}

// escapeMD replaces the characters that break a single table cell.
// Pipes split cells; backticks would close the inline-code rendering.
func escapeMD(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

func humanSize(n int64) string {
	if n == 0 {
		return ""
	}
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case n >= GB:
		return fmt.Sprintf("%.1f GB", float64(n)/GB)
	case n >= MB:
		return fmt.Sprintf("%.1f MB", float64(n)/MB)
	case n >= KB:
		return fmt.Sprintf("%.1f KB", float64(n)/KB)
	default:
		return fmt.Sprintf("%d B", n)
	}
}
