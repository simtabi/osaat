package reporters

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/simtabi/osaat/internal/audit"
)

// sampleRecords returns a small fixed set used by every reporter test.
func sampleRecords() []audit.AppRecord {
	return []audit.AppRecord{
		{Name: "Zebra", BundleID: "com.example.zebra", Version: "9.9", Source: audit.SourceUnknown, SigningStatus: audit.SigningUnknown, SizeBytes: 0},
		{Name: "alpha", BundleID: "com.example.alpha", Version: "1.0", Source: audit.SourceAppStore, AppStoreID: "12345", SigningStatus: audit.SigningSigned, SigningTeam: "ALPHA1234", Author: "Alpha Inc.", ReinstallCmd: "mas install 12345", SizeBytes: 4_500_000},
		{Name: "Mango", BundleID: "com.example.mango", Version: "2.3.4", Source: audit.SourceBrewCask, ReinstallCmd: "brew install --cask mango", SizeBytes: 100_000_000, CollectorNotes: []string{"quarantined by Safari"}},
	}
}

func TestCSVReporterShape(t *testing.T) {
	r := NewCSVReporter()
	var buf bytes.Buffer
	if err := r.Write(sampleRecords(), &buf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	rows, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("expected 1 header + 3 records, got %d rows: %v", len(rows), rows)
	}
	if rows[0][0] != "name" || rows[0][2] != "source" {
		t.Errorf("header columns wrong: %v", rows[0])
	}
	// Sorted: alpha, Mango, Zebra (case-insensitive).
	wantOrder := []string{"alpha", "Mango", "Zebra"}
	for i, want := range wantOrder {
		if rows[i+1][0] != want {
			t.Errorf("row %d name: got %q, want %q", i+1, rows[i+1][0], want)
		}
	}
}

func TestMarkdownReporterShape(t *testing.T) {
	r := &MarkdownReporter{Now: func() time.Time { return time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC) }}
	var buf bytes.Buffer
	if err := r.Write(sampleRecords(), &buf); err != nil {
		t.Fatalf("Write: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"# Application inventory",
		"## Summary",
		"## Applications",
		"| Name | Version |",
		"alpha",
		"Mango",
		"Zebra",
		"`app_store`",
		"`homebrew_cask`",
		"quarantined by Safari", // notes section
	} {
		if !strings.Contains(out, want) {
			t.Errorf("markdown output missing %q", want)
		}
	}
	if r.Format() != "markdown" {
		t.Errorf("Format() = %q", r.Format())
	}
}

func TestHTMLReporterShape(t *testing.T) {
	r := &HTMLReporter{Now: func() time.Time { return time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC) }}
	var buf bytes.Buffer
	if err := r.Write(sampleRecords(), &buf); err != nil {
		t.Fatalf("Write: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"<!doctype html>",
		"Application inventory",
		"alpha",
		"Mango",
		"Zebra",
		`class="src-app_store"`,
		`class="src-homebrew_cask`,
		`id="filter"`, // search input
	} {
		if !strings.Contains(out, want) {
			t.Errorf("html output missing %q", want)
		}
	}
	if r.Format() != "html" {
		t.Errorf("Format() = %q", r.Format())
	}
}

func TestHumanSize(t *testing.T) {
	cases := map[int64]string{
		0:            "",
		512:          "512 B",
		2048:         "2.0 KB",
		3_500_000:    "3.3 MB",
		15_000_000_000: "14.0 GB",
	}
	for in, want := range cases {
		if got := humanSize(in); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", in, got, want)
		}
	}
}
