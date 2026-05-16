package restore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/simtabi/osaat/internal/audit"
)

// WriteAll writes Brewfile, mas-apps.txt, and RESTORE.md to outDir.
// Files that would be empty (e.g. no App Store apps with IDs) are
// still written with their header so the user has a placeholder to
// edit by hand. Returns the list of paths that were written.
func WriteAll(records []audit.AppRecord, outDir string) ([]string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}

	var written []string
	type artifact struct {
		filename string
		write    func(io.Writer) error
	}
	artifacts := []artifact{
		{"Brewfile", func(w io.Writer) error { return WriteBrewfile(records, w) }},
		{"mas-apps.txt", func(w io.Writer) error { return WriteMasList(records, w) }},
		{"RESTORE.md", func(w io.Writer) error { return WriteRestoreDoc(records, w) }},
	}

	for _, a := range artifacts {
		path := filepath.Join(outDir, a.filename)
		f, err := os.Create(path)
		if err != nil {
			return written, fmt.Errorf("create %s: %w", path, err)
		}
		if err := a.write(f); err != nil {
			_ = f.Close()
			return written, fmt.Errorf("write %s: %w", path, err)
		}
		if err := f.Close(); err != nil {
			return written, fmt.Errorf("close %s: %w", path, err)
		}
		written = append(written, path)
	}
	return written, nil
}

// WriteRestoreDoc emits RESTORE.md — the human checklist for apps
// that aren't covered by `brew bundle install` or `mas install`.
// Apps are grouped by source so the user can work through each
// section. Each entry includes the vendor URL when known, the
// original download URL (from kMDItemWhereFroms / quarantine), and
// any collector notes.
func WriteRestoreDoc(records []audit.AppRecord, w io.Writer) error {
	hostname, _ := os.Hostname()

	var b strings.Builder
	fmt.Fprintf(&b, "# Manual restore checklist\n\n")
	fmt.Fprintf(&b, "Generated %s on `%s`. Use `Brewfile` and `mas-apps.txt`\n", time.Now().UTC().Format(time.RFC3339), hostname)
	fmt.Fprintf(&b, "first to cover the automated cases; this file lists the long tail.\n\n")

	// Bucket records by source. Anything covered by brew/mas is excluded.
	byBucket := map[string][]audit.AppRecord{}
	for _, r := range records {
		switch r.Source {
		case audit.SourceBrewCask, audit.SourceBrewFormula:
			continue
		case audit.SourceAppStore:
			if r.AppStoreID != "" {
				continue // already in mas-apps.txt
			}
			byBucket["App Store (no mas ID — install manually)"] = append(byBucket["App Store (no mas ID — install manually)"], r)
		case audit.SourcePkg:
			byBucket["Installer packages (`.pkg`)"] = append(byBucket["Installer packages (`.pkg`)"], r)
		case audit.SourceDMG:
			byBucket["Direct download (`.dmg`, vendor websites)"] = append(byBucket["Direct download (`.dmg`, vendor websites)"], r)
		case audit.SourceSystem, audit.SourceSandbox:
			// system apps come back automatically; skip
			continue
		default:
			byBucket["Unknown source"] = append(byBucket["Unknown source"], r)
		}
	}

	// Render buckets in a stable order.
	bucketOrder := []string{
		"App Store (no mas ID — install manually)",
		"Direct download (`.dmg`, vendor websites)",
		"Installer packages (`.pkg`)",
		"Unknown source",
	}

	wrote := false
	for _, name := range bucketOrder {
		recs := byBucket[name]
		if len(recs) == 0 {
			continue
		}
		wrote = true
		sort.SliceStable(recs, func(i, j int) bool {
			return strings.ToLower(recs[i].Name) < strings.ToLower(recs[j].Name)
		})
		fmt.Fprintf(&b, "## %s\n\n", name)
		for _, r := range recs {
			writeRestoreEntry(&b, r)
		}
	}

	if !wrote {
		b.WriteString("Nothing to install manually — every app is covered by Brewfile or mas-apps.txt.\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func writeRestoreEntry(b *strings.Builder, r audit.AppRecord) {
	title := r.Name
	if r.Version != "" {
		title = fmt.Sprintf("%s (%s)", title, r.Version)
	}
	fmt.Fprintf(b, "- [ ] **%s**\n", title)
	if r.Author != "" {
		fmt.Fprintf(b, "  - Author: %s\n", r.Author)
	}
	if r.BundleID != "" {
		fmt.Fprintf(b, "  - Bundle: `%s`\n", r.BundleID)
	}
	if r.DownloadURL != "" {
		fmt.Fprintf(b, "  - Download URL (from this machine): %s\n", r.DownloadURL)
	}
	if r.VendorURL != "" {
		fmt.Fprintf(b, "  - Vendor: %s\n", r.VendorURL)
	}
	if r.ReinstallCmd != "" {
		fmt.Fprintf(b, "  - Reinstall: `%s`\n", r.ReinstallCmd)
	}
	if len(r.CollectorNotes) > 0 {
		fmt.Fprintf(b, "  - Notes: %s\n", strings.Join(r.CollectorNotes, "; "))
	}
	fmt.Fprintln(b)
}
