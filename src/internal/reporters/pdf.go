package reporters

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/version"
)

// PDFReporter renders the inventory as a single-file PDF: title +
// metadata + per-source summary + a paginated row table. Uses
// go-pdf/fpdf which is pure Go and ships with the Helvetica core font,
// so no external font files are needed.
type PDFReporter struct {
	Now func() time.Time
}

// NewPDFReporter returns a PDF reporter that uses the real wall clock.
func NewPDFReporter() *PDFReporter {
	return &PDFReporter{Now: func() time.Time { return time.Now().UTC() }}
}

// Format implements Reporter.
func (r *PDFReporter) Format() string { return "pdf" }

// Write implements Reporter.
func (r *PDFReporter) Write(records []audit.AppRecord, w io.Writer) error {
	sorted := make([]audit.AppRecord, len(records))
	copy(sorted, records)
	sort.SliceStable(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	hostname, _ := os.Hostname()
	now := r.Now()

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("osaat application inventory", false)
	pdf.SetCreator("osaat "+version.Version, false)

	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Helvetica", "I", 8)
		pdf.SetTextColor(110, 110, 110)
		pdf.CellFormat(0, 10, fmt.Sprintf("osaat %s — page %d", version.Version, pdf.PageNo()), "", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	})

	pdf.AddPage()
	r.writeHeader(pdf, hostname, now, len(sorted))
	r.writeSummary(pdf, sorted)
	r.writeTable(pdf, sorted)

	return pdf.Output(w)
}

func (r *PDFReporter) writeHeader(pdf *fpdf.Fpdf, hostname string, now time.Time, count int) {
	pdf.SetFont("Helvetica", "B", 18)
	pdf.Cell(0, 10, "osaat application inventory")
	pdf.Ln(12)

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(80, 80, 80)
	for _, line := range []string{
		"Generated:    " + now.Format(time.RFC3339),
		"Host:         " + hostname + " (" + runtime.GOOS + "/" + runtime.GOARCH + ")",
		"Tool:         osaat " + version.Version,
		"Applications: " + fmt.Sprintf("%d", count),
	} {
		pdf.Cell(0, 5, line)
		pdf.Ln(5)
	}
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(4)
}

func (r *PDFReporter) writeSummary(pdf *fpdf.Fpdf, records []audit.AppRecord) {
	bySource := map[audit.Source]int{}
	for _, rec := range records {
		bySource[rec.Source]++
	}

	pdf.SetFont("Helvetica", "B", 11)
	pdf.Cell(0, 6, "By source")
	pdf.Ln(7)

	pdf.SetFont("Helvetica", "", 9)
	for _, src := range orderedSources() {
		if n, ok := bySource[src]; ok && n > 0 {
			pdf.CellFormat(60, 5, string(src), "", 0, "L", false, 0, "")
			pdf.CellFormat(20, 5, fmt.Sprintf("%d", n), "", 0, "R", false, 0, "")
			pdf.Ln(5)
		}
	}
	pdf.Ln(4)
}

// PDF column widths in mm (A4 portrait usable width is ~190mm).
var pdfColumns = []struct {
	width float64
	label string
	get   func(audit.AppRecord) string
}{
	{55, "Name", func(r audit.AppRecord) string { return r.Name }},
	{20, "Version", func(r audit.AppRecord) string { return r.Version }},
	{30, "Source", func(r audit.AppRecord) string { return string(r.Source) }},
	{22, "Signing", func(r audit.AppRecord) string { return string(r.SigningStatus) }},
	{63, "Reinstall / Author", func(r audit.AppRecord) string {
		if r.ReinstallCmd != "" {
			return r.ReinstallCmd
		}
		return r.Author
	}},
}

func (r *PDFReporter) writeTable(pdf *fpdf.Fpdf, records []audit.AppRecord) {
	pdf.SetFont("Helvetica", "B", 11)
	pdf.Cell(0, 6, "Applications")
	pdf.Ln(7)

	pdf.SetFillColor(60, 60, 60)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 8)
	for _, col := range pdfColumns {
		pdf.CellFormat(col.width, 6, col.label, "", 0, "L", true, 0, "")
	}
	pdf.Ln(6)

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 8)

	for i, rec := range records {
		// Zebra stripe.
		fill := i%2 == 0
		if fill {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		for _, col := range pdfColumns {
			pdf.CellFormat(col.width, 5, truncateForPDF(col.get(rec), col.width), "", 0, "L", true, 0, "")
		}
		pdf.Ln(5)
	}
}

// truncateForPDF clips a string to roughly fit into width mm at the
// table's font size. Helvetica 8pt averages ~1.8mm per character.
func truncateForPDF(s string, widthMM float64) string {
	limit := int(widthMM / 1.8)
	if limit < 4 {
		limit = 4
	}
	if len(s) <= limit {
		return s
	}
	return s[:limit-1] + "…"
}
