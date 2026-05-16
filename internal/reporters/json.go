package reporters

import (
	"encoding/json"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/version"
)

// JSONReporter emits a top-level object with schema + host metadata +
// the record array. This shape lets readers verify the schema field
// before deserializing.
type JSONReporter struct {
	Now func() time.Time // injectable for deterministic tests
}

// NewJSONReporter returns a reporter that uses the real wall clock.
func NewJSONReporter() *JSONReporter {
	return &JSONReporter{Now: func() time.Time { return time.Now().UTC() }}
}

// Format implements Reporter.
func (r *JSONReporter) Format() string { return "json" }

// Write implements Reporter.
func (r *JSONReporter) Write(records []audit.AppRecord, w io.Writer) error {
	sorted := make([]audit.AppRecord, len(records))
	copy(sorted, records)
	sort.SliceStable(sorted, func(i, j int) bool {
		li, lj := strings.ToLower(sorted[i].Name), strings.ToLower(sorted[j].Name)
		if li != lj {
			return li < lj
		}
		return sorted[i].BundleID < sorted[j].BundleID
	})

	hostname, _ := os.Hostname()

	doc := jsonReport{
		Schema:      audit.SchemaReportV1,
		GeneratedAt: r.Now(),
		Host: hostInfo{
			Hostname: hostname,
			OS:       runtime.GOOS,
			Arch:     runtime.GOARCH,
		},
		Tool: toolInfo{
			Name:    "osaat",
			Version: version.Version,
			Commit:  version.Commit,
		},
		Records: sorted,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(doc)
}

type jsonReport struct {
	Schema      string            `json:"schema"`
	GeneratedAt time.Time         `json:"generated_at"`
	Host        hostInfo          `json:"host"`
	Tool        toolInfo          `json:"tool"`
	Records     []audit.AppRecord `json:"records"`
}

type hostInfo struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
}

type toolInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}
