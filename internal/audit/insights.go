package audit

import (
	"fmt"
	"time"
)

// MarkForgotten flags every record whose LastUsedAt is older than
// cutoffMonths months. Records with no LastUsedAt (apps that have
// never been opened — Spotlight returned "(null)") are flagged only
// when their InstalledAt is also older than cutoffMonths.
//
// The flag manifests as:
//   - A "forgotten: <reason>" entry in CollectorNotes.
//
// Returns the number of records flagged.
func MarkForgotten(records []AppRecord, cutoffMonths int, now time.Time) int {
	if cutoffMonths <= 0 {
		return 0
	}
	cutoff := now.AddDate(0, -cutoffMonths, 0)
	var flagged int
	for i := range records {
		if !needsForgottenCheck(records[i]) {
			continue
		}
		switch {
		case records[i].LastUsedAt != nil && records[i].LastUsedAt.Before(cutoff):
			records[i].Note(fmt.Sprintf("forgotten: last used %s", records[i].LastUsedAt.Format("2006-01-02")))
			flagged++
		case records[i].LastUsedAt == nil && records[i].InstalledAt != nil && records[i].InstalledAt.Before(cutoff):
			records[i].Note("forgotten: never opened since install")
			flagged++
		}
	}
	return flagged
}

// needsForgottenCheck excludes records that shouldn't be flagged —
// system apps, sandbox apps, and Homebrew formulae are always treated
// as "needed" regardless of when they were last used.
func needsForgottenCheck(r AppRecord) bool {
	switch r.Source {
	case SourceSystem, SourceSandbox, SourceBrewFormula:
		return false
	}
	return true
}
