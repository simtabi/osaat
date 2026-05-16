// Package diff compares two AppRecord sets and produces a structured
// Result describing what was added, removed, and modified.
package diff

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/simtabi/osaat/internal/audit"
)

// SchemaDiffV1 is the value of the "schema" field at the top of a
// diff result emitted in JSON form.
const SchemaDiffV1 = "osaat.diff/v1"

// Result is the canonical comparison output. Categories are stable in
// order across runs to keep the JSON form diffable itself.
type Result struct {
	Schema   string            `json:"schema"`
	OldFile  string            `json:"old_file,omitempty"`
	NewFile  string            `json:"new_file,omitempty"`
	Added    []audit.AppRecord `json:"added"`
	Removed  []audit.AppRecord `json:"removed"`
	Modified []Modification    `json:"modified"`
}

// Modification describes a record that exists in both reports with
// at least one differing field.
type Modification struct {
	Name   string          `json:"name"`
	Key    string          `json:"match_key"`
	Fields []FieldDiff     `json:"fields"`
	Old    audit.AppRecord `json:"old"`
	New    audit.AppRecord `json:"new"`
}

// FieldDiff is a single (field name, before, after) tuple.
type FieldDiff struct {
	Field string `json:"field"`
	Old   string `json:"old"`
	New   string `json:"new"`
}

// IsClean reports whether the two record sets are equivalent under
// the diff comparison.
func (r Result) IsClean() bool {
	return len(r.Added) == 0 && len(r.Removed) == 0 && len(r.Modified) == 0
}

// Compare returns the diff between two record sets. Records are
// matched by BundleID first, then PkgID, then by Name+Version when
// neither identifier is present.
func Compare(oldRecords, newRecords []audit.AppRecord) Result {
	oldIdx := index(oldRecords)
	newIdx := index(newRecords)

	var added, removed []audit.AppRecord
	var modified []Modification

	for k, n := range newIdx {
		o, ok := oldIdx[k]
		if !ok {
			added = append(added, *n)
			continue
		}
		fields := diffFields(*o, *n)
		if len(fields) > 0 {
			modified = append(modified, Modification{
				Name: n.Name, Key: k, Fields: fields, Old: *o, New: *n,
			})
		}
	}
	for k, o := range oldIdx {
		if _, ok := newIdx[k]; !ok {
			removed = append(removed, *o)
		}
	}

	sortByName(added)
	sortByName(removed)
	sort.SliceStable(modified, func(i, j int) bool {
		return strings.ToLower(modified[i].Name) < strings.ToLower(modified[j].Name)
	})

	return Result{
		Schema:   SchemaDiffV1,
		Added:    added,
		Removed:  removed,
		Modified: modified,
	}
}

func index(records []audit.AppRecord) map[string]*audit.AppRecord {
	m := make(map[string]*audit.AppRecord, len(records))
	for i := range records {
		m[key(&records[i])] = &records[i]
	}
	return m
}

func key(r *audit.AppRecord) string {
	if r.BundleID != "" {
		return "bid:" + r.BundleID
	}
	if r.PkgID != "" {
		return "pid:" + r.PkgID
	}
	return "nv:" + r.Name + "@" + r.Version
}

func diffFields(a, b audit.AppRecord) []FieldDiff {
	var diffs []FieldDiff
	add := func(name, before, after string) {
		if before != after {
			diffs = append(diffs, FieldDiff{Field: name, Old: before, New: after})
		}
	}
	add("version", a.Version, b.Version)
	add("source", string(a.Source), string(b.Source))
	add("signing_status", string(a.SigningStatus), string(b.SigningStatus))
	add("author", a.Author, b.Author)
	add("reinstall_cmd", a.ReinstallCmd, b.ReinstallCmd)
	return diffs
}

func sortByName(records []audit.AppRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		return strings.ToLower(records[i].Name) < strings.ToLower(records[j].Name)
	})
}

// reportFile mirrors the JSON envelope written by the JSON reporter.
// We only care about the records slice when loading for a diff.
type reportFile struct {
	Schema  string            `json:"schema"`
	Records []audit.AppRecord `json:"records"`
}

// LoadReport reads a report.json file produced by `osaat scan` and
// returns its records. Unknown schemas surface as a non-fatal warning
// on stderr; the records are still returned.
func LoadReport(path string) ([]audit.AppRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var rep reportFile
	if err := json.NewDecoder(f).Decode(&rep); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if rep.Schema != "" && rep.Schema != audit.SchemaReportV1 {
		fmt.Fprintf(os.Stderr, "warning: %s reports schema %q; expected %q\n", path, rep.Schema, audit.SchemaReportV1)
	}
	return rep.Records, nil
}

// WriteText writes a human-readable summary of the diff to w.
func WriteText(r Result, w io.Writer) error {
	var b strings.Builder
	fmt.Fprintf(&b, "osaat diff")
	if r.OldFile != "" && r.NewFile != "" {
		fmt.Fprintf(&b, " %s vs %s", r.OldFile, r.NewFile)
	}
	fmt.Fprintln(&b)

	fmt.Fprintf(&b, "  +%d added, -%d removed, ~%d modified\n\n", len(r.Added), len(r.Removed), len(r.Modified))

	if len(r.Added) > 0 {
		fmt.Fprintf(&b, "Added:\n")
		for _, rec := range r.Added {
			fmt.Fprintf(&b, "  + %s%s [%s]\n", rec.Name, verSuffix(rec.Version), rec.Source)
		}
		fmt.Fprintln(&b)
	}
	if len(r.Removed) > 0 {
		fmt.Fprintf(&b, "Removed:\n")
		for _, rec := range r.Removed {
			fmt.Fprintf(&b, "  - %s%s [%s]\n", rec.Name, verSuffix(rec.Version), rec.Source)
		}
		fmt.Fprintln(&b)
	}
	if len(r.Modified) > 0 {
		fmt.Fprintf(&b, "Modified:\n")
		for _, m := range r.Modified {
			fmt.Fprintf(&b, "  ~ %s\n", m.Name)
			for _, fd := range m.Fields {
				fmt.Fprintf(&b, "      %s: %q -> %q\n", fd.Field, fd.Old, fd.New)
			}
		}
		fmt.Fprintln(&b)
	}
	if r.IsClean() {
		fmt.Fprintf(&b, "No differences.\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// WriteJSON writes the diff as the canonical Result envelope.
func WriteJSON(r Result, w io.Writer) error {
	if r.Added == nil {
		r.Added = []audit.AppRecord{}
	}
	if r.Removed == nil {
		r.Removed = []audit.AppRecord{}
	}
	if r.Modified == nil {
		r.Modified = []Modification{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(r)
}

// WriteMarkdown writes the diff as a GitHub-flavored Markdown
// document with one table per category.
func WriteMarkdown(r Result, w io.Writer) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Inventory diff\n\n")
	if r.OldFile != "" {
		fmt.Fprintf(&b, "- Old: `%s`\n", r.OldFile)
	}
	if r.NewFile != "" {
		fmt.Fprintf(&b, "- New: `%s`\n", r.NewFile)
	}
	fmt.Fprintf(&b, "- Added: %d • Removed: %d • Modified: %d\n\n",
		len(r.Added), len(r.Removed), len(r.Modified))

	if len(r.Added) > 0 {
		fmt.Fprintf(&b, "## Added\n\n| Name | Version | Source |\n|---|---|---|\n")
		for _, rec := range r.Added {
			fmt.Fprintf(&b, "| %s | %s | `%s` |\n", rec.Name, rec.Version, rec.Source)
		}
		fmt.Fprintln(&b)
	}
	if len(r.Removed) > 0 {
		fmt.Fprintf(&b, "## Removed\n\n| Name | Version | Source |\n|---|---|---|\n")
		for _, rec := range r.Removed {
			fmt.Fprintf(&b, "| %s | %s | `%s` |\n", rec.Name, rec.Version, rec.Source)
		}
		fmt.Fprintln(&b)
	}
	if len(r.Modified) > 0 {
		fmt.Fprintf(&b, "## Modified\n\n| Name | Field | Old | New |\n|---|---|---|---|\n")
		for _, m := range r.Modified {
			for _, fd := range m.Fields {
				fmt.Fprintf(&b, "| %s | `%s` | `%s` | `%s` |\n", m.Name, fd.Field, fd.Old, fd.New)
			}
		}
		fmt.Fprintln(&b)
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func verSuffix(v string) string {
	if v == "" {
		return ""
	}
	return " (" + v + ")"
}
