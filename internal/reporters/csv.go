package reporters

import (
	"encoding/csv"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/simtabi/osaat/internal/audit"
)

// CSVReporter writes a flat CSV with one row per record. Column order
// is stable across runs so diffs of the CSV are meaningful.
type CSVReporter struct{}

// NewCSVReporter returns a CSV reporter.
func NewCSVReporter() *CSVReporter { return &CSVReporter{} }

// Format implements Reporter.
func (r *CSVReporter) Format() string { return "csv" }

// csvHeader is the canonical column order. Reordering breaks downstream
// spreadsheet consumers, so changes should be additive (append-only).
var csvHeader = []string{
	"name",
	"version",
	"source",
	"author",
	"vendor_url",
	"download_url",
	"bundle_id",
	"pkg_id",
	"path",
	"size_bytes",
	"installed_at",
	"last_used_at",
	"signing_status",
	"signing_team",
	"apple_silicon",
	"reinstall_cmd",
	"app_store_id",
	"notes",
}

// Write implements Reporter.
func (r *CSVReporter) Write(records []audit.AppRecord, w io.Writer) error {
	sorted := make([]audit.AppRecord, len(records))
	copy(sorted, records)
	sort.SliceStable(sorted, func(i, j int) bool {
		li, lj := strings.ToLower(sorted[i].Name), strings.ToLower(sorted[j].Name)
		if li != lj {
			return li < lj
		}
		return sorted[i].BundleID < sorted[j].BundleID
	})

	cw := csv.NewWriter(w)
	if err := cw.Write(csvHeader); err != nil {
		return err
	}
	for _, rec := range sorted {
		if err := cw.Write(csvRow(rec)); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func csvRow(r audit.AppRecord) []string {
	return []string{
		r.Name,
		r.Version,
		string(r.Source),
		r.Author,
		r.VendorURL,
		r.DownloadURL,
		r.BundleID,
		r.PkgID,
		r.Path,
		intIfNonZero(r.SizeBytes),
		timePtrString(r.InstalledAt),
		timePtrString(r.LastUsedAt),
		string(r.SigningStatus),
		r.SigningTeam,
		boolPtr(r.AppleSilicon),
		r.ReinstallCmd,
		r.AppStoreID,
		strings.Join(r.CollectorNotes, "; "),
	}
}

func intIfNonZero(n int64) string {
	if n == 0 {
		return ""
	}
	return strconv.FormatInt(n, 10)
}

func timeIfNonZero(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// timePtrString formats a *time.Time as RFC 3339 UTC, or "" when nil.
func timePtrString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func boolPtr(b *bool) string {
	if b == nil {
		return ""
	}
	if *b {
		return "true"
	}
	return "false"
}
