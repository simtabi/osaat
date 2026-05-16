package reporters

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/simtabi/osaat/internal/audit"
)

func TestJSONReporterSortAndShape(t *testing.T) {
	fixed := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	r := &JSONReporter{Now: func() time.Time { return fixed }}

	records := []audit.AppRecord{
		{Name: "Zebra", BundleID: "com.example.zebra", Source: audit.SourceUnknown},
		{Name: "alpha", BundleID: "com.example.alpha", Source: audit.SourceAppStore},
		{Name: "Mango", BundleID: "com.example.mango.a", Source: audit.SourceBrewCask},
		{Name: "Mango", BundleID: "com.example.mango.b", Source: audit.SourceBrewCask},
	}

	var buf bytes.Buffer
	if err := r.Write(records, &buf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	var got struct {
		Schema      string            `json:"schema"`
		GeneratedAt time.Time         `json:"generated_at"`
		Records     []audit.AppRecord `json:"records"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, buf.String())
	}

	if got.Schema != audit.SchemaReportV1 {
		t.Errorf("schema: got %q, want %q", got.Schema, audit.SchemaReportV1)
	}
	if !got.GeneratedAt.Equal(fixed) {
		t.Errorf("generated_at: got %v, want %v", got.GeneratedAt, fixed)
	}
	if len(got.Records) != 4 {
		t.Fatalf("records: got %d, want 4", len(got.Records))
	}

	// Records should be sorted: alpha, Mango (com.example.mango.a),
	// Mango (com.example.mango.b), Zebra — case-insensitive by name,
	// tie-broken by bundle id.
	wantOrder := []string{
		"com.example.alpha",
		"com.example.mango.a",
		"com.example.mango.b",
		"com.example.zebra",
	}
	for i, want := range wantOrder {
		if got.Records[i].BundleID != want {
			t.Errorf("record %d: got BundleID %q, want %q", i, got.Records[i].BundleID, want)
		}
	}
}

func TestJSONReporterFormat(t *testing.T) {
	r := NewJSONReporter()
	if r.Format() != "json" {
		t.Errorf("Format() = %q, want json", r.Format())
	}
}
