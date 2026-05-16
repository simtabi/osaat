package audit

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNoteDeduplicates(t *testing.T) {
	r := AppRecord{}
	r.Note("unsigned binary")
	r.Note("unsigned binary")
	r.Note("quarantined by Safari")
	if got := len(r.CollectorNotes); got != 2 {
		t.Fatalf("expected 2 unique notes, got %d: %v", got, r.CollectorNotes)
	}
}

func TestAppRecordJSONOmitsEmpty(t *testing.T) {
	r := AppRecord{Name: "TestApp", Source: SourceUnknown}
	out, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(out)
	for _, key := range []string{"bundle_id", "pkg_id", "version", "vendor_url", "size_bytes", "apple_silicon"} {
		if strings.Contains(s, "\""+key+"\"") {
			t.Errorf("expected %q to be omitted from JSON; got: %s", key, s)
		}
	}
	if !strings.Contains(s, "\"name\":\"TestApp\"") {
		t.Errorf("expected name=TestApp in JSON; got: %s", s)
	}
}
