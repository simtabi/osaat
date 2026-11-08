package audit

import (
	"strings"
	"testing"
	"time"
)

func TestMarkForgottenLastUsed(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	cutoff := 6

	tp := func(t time.Time) *time.Time { return &t }

	records := []AppRecord{
		{Name: "Recent", LastUsedAt: tp(now.AddDate(0, -1, 0)), Source: SourceDMG},
		{Name: "Old", LastUsedAt: tp(now.AddDate(0, -8, 0)), Source: SourceDMG},
		{Name: "System", LastUsedAt: tp(now.AddDate(0, -12, 0)), Source: SourceSystem},
		{Name: "Borderline", LastUsedAt: tp(now.AddDate(0, -6, 0).Add(time.Hour)), Source: SourceDMG},
	}
	got := MarkForgotten(records, cutoff, now)
	if got != 1 {
		t.Errorf("flagged: got %d, want 1 (Old only)", got)
	}
	if !contains(records[1].CollectorNotes, "forgotten:") {
		t.Errorf("Old not flagged: %+v", records[1].CollectorNotes)
	}
	if len(records[0].CollectorNotes) != 0 || len(records[2].CollectorNotes) != 0 || len(records[3].CollectorNotes) != 0 {
		t.Errorf("wrong records flagged: %+v", records)
	}
}

func TestMarkForgottenNeverOpened(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	tp := func(t time.Time) *time.Time { return &t }
	records := []AppRecord{
		{Name: "OldInstall", InstalledAt: tp(now.AddDate(0, -12, 0)), Source: SourceDMG},
		{Name: "FreshInstall", InstalledAt: tp(now.AddDate(0, -1, 0)), Source: SourceDMG},
		{Name: "OldInstallButUsed", InstalledAt: tp(now.AddDate(0, -12, 0)), LastUsedAt: tp(now.AddDate(0, -1, 0)), Source: SourceDMG},
	}
	got := MarkForgotten(records, 6, now)
	if got != 1 {
		t.Errorf("flagged: got %d, want 1 (OldInstall only)", got)
	}
	if !contains(records[0].CollectorNotes, "never opened") {
		t.Errorf("OldInstall not flagged correctly: %+v", records[0].CollectorNotes)
	}
}

func TestMarkForgottenZeroCutoffNoop(t *testing.T) {
	old := time.Now().AddDate(0, -100, 0)
	records := []AppRecord{
		{Name: "X", LastUsedAt: &old, Source: SourceDMG},
	}
	if got := MarkForgotten(records, 0, time.Now()); got != 0 {
		t.Errorf("zero cutoff should be no-op; flagged %d", got)
	}
}

func contains(strs []string, needle string) bool {
	for _, s := range strs {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}
