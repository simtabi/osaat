package diff

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/simtabi/osaat/internal/audit"
)

func TestCompareDetectsAddedRemovedModified(t *testing.T) {
	oldRecs := []audit.AppRecord{
		{Name: "Keep", BundleID: "com.example.keep", Version: "1.0", Source: audit.SourceAppStore},
		{Name: "ChangeVersion", BundleID: "com.example.cv", Version: "1.0", Source: audit.SourceBrewCask},
		{Name: "ChangeSource", BundleID: "com.example.cs", Version: "1.0", Source: audit.SourceDMG},
		{Name: "RemovedApp", BundleID: "com.example.removed", Version: "1.0", Source: audit.SourceAppStore},
	}
	newRecs := []audit.AppRecord{
		{Name: "Keep", BundleID: "com.example.keep", Version: "1.0", Source: audit.SourceAppStore},
		{Name: "ChangeVersion", BundleID: "com.example.cv", Version: "1.1", Source: audit.SourceBrewCask},
		{Name: "ChangeSource", BundleID: "com.example.cs", Version: "1.0", Source: audit.SourceBrewCask},
		{Name: "AddedApp", BundleID: "com.example.added", Version: "2.0", Source: audit.SourceAppStore},
	}

	r := Compare(oldRecs, newRecs)

	if len(r.Added) != 1 || r.Added[0].Name != "AddedApp" {
		t.Errorf("added: %+v", r.Added)
	}
	if len(r.Removed) != 1 || r.Removed[0].Name != "RemovedApp" {
		t.Errorf("removed: %+v", r.Removed)
	}
	if len(r.Modified) != 2 {
		t.Fatalf("expected 2 modified, got %d: %+v", len(r.Modified), r.Modified)
	}
	// Sorted alphabetically by name: ChangeSource, ChangeVersion.
	if r.Modified[0].Name != "ChangeSource" || r.Modified[1].Name != "ChangeVersion" {
		t.Errorf("modified order wrong: %v", []string{r.Modified[0].Name, r.Modified[1].Name})
	}
	cs := r.Modified[0]
	if len(cs.Fields) != 1 || cs.Fields[0].Field != "source" {
		t.Errorf("ChangeSource fields: %+v", cs.Fields)
	}
	if cs.Fields[0].Old != "dmg_or_direct" || cs.Fields[0].New != "homebrew_cask" {
		t.Errorf("ChangeSource diff: %+v", cs.Fields[0])
	}
}

func TestCompareNoChanges(t *testing.T) {
	recs := []audit.AppRecord{{Name: "A", BundleID: "com.a", Version: "1"}}
	r := Compare(recs, recs)
	if !r.IsClean() {
		t.Errorf("expected clean, got: %+v", r)
	}
}

func TestKeyFallback(t *testing.T) {
	cases := []struct {
		rec  audit.AppRecord
		want string
	}{
		{audit.AppRecord{BundleID: "com.a"}, "bid:com.a"},
		{audit.AppRecord{PkgID: "pkg-a"}, "pid:pkg-a"},
		{audit.AppRecord{Name: "Foo", Version: "1.0"}, "nv:Foo@1.0"},
		{audit.AppRecord{BundleID: "com.a", PkgID: "pkg-a"}, "bid:com.a"}, // BundleID wins
	}
	for _, tc := range cases {
		got := key(&tc.rec)
		if got != tc.want {
			t.Errorf("key(%+v) = %q, want %q", tc.rec, got, tc.want)
		}
	}
}

func TestWriteTextSummary(t *testing.T) {
	r := Result{
		Schema:  SchemaDiffV1,
		Added:   []audit.AppRecord{{Name: "NewApp", Version: "1.0", Source: audit.SourceAppStore}},
		Removed: []audit.AppRecord{{Name: "OldApp", Version: "9.0", Source: audit.SourceDMG}},
		Modified: []Modification{{
			Name:   "Changed",
			Fields: []FieldDiff{{Field: "version", Old: "1.0", New: "1.1"}},
		}},
	}
	var buf bytes.Buffer
	if err := WriteText(r, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"+1 added, -1 removed, ~1 modified",
		"+ NewApp (1.0) [app_store]",
		"- OldApp (9.0) [dmg_or_direct]",
		"~ Changed",
		`version: "1.0" -> "1.1"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestWriteJSONRoundTrip(t *testing.T) {
	r := Result{
		Schema:  SchemaDiffV1,
		Added:   []audit.AppRecord{{Name: "NewApp"}},
		Removed: nil, // exercise the nil → [] conversion
	}
	var buf bytes.Buffer
	if err := WriteJSON(r, &buf); err != nil {
		t.Fatal(err)
	}
	var back Result
	if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, buf.String())
	}
	if back.Schema != SchemaDiffV1 {
		t.Errorf("schema: %q", back.Schema)
	}
	if len(back.Added) != 1 || back.Added[0].Name != "NewApp" {
		t.Errorf("added: %+v", back.Added)
	}
	if back.Removed == nil {
		t.Error("Removed should be [] not nil after JSON round-trip")
	}
}
