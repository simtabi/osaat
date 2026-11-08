package reporters

import (
	"html/template"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/version"
)

// HTMLReporter writes a single self-contained HTML file with an inline
// stylesheet and vanilla JavaScript for column sorting.
type HTMLReporter struct {
	Now func() time.Time
}

// NewHTMLReporter returns a reporter that uses the real wall clock.
func NewHTMLReporter() *HTMLReporter {
	return &HTMLReporter{Now: func() time.Time { return time.Now().UTC() }}
}

// Format implements Reporter.
func (r *HTMLReporter) Format() string { return "html" }

type htmlContext struct {
	GeneratedAt string
	Hostname    string
	OS          string
	Tool        string
	Count       int
	Records     []htmlRow
}

type htmlRow struct {
	Name          string
	Version       string
	Source        string
	Signing       string
	SigningTeam   string
	Author        string
	BundleID      string
	Size          string
	InstalledAt   string
	DownloadURL   string
	ReinstallCmd  string
	Notes         string
}

// Write implements Reporter.
func (r *HTMLReporter) Write(records []audit.AppRecord, w io.Writer) error {
	sorted := make([]audit.AppRecord, len(records))
	copy(sorted, records)
	sort.SliceStable(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})

	rows := make([]htmlRow, 0, len(sorted))
	for _, rec := range sorted {
		rows = append(rows, htmlRow{
			Name:         rec.Name,
			Version:      rec.Version,
			Source:       string(rec.Source),
			Signing:      string(rec.SigningStatus),
			SigningTeam:  rec.SigningTeam,
			Author:       rec.Author,
			BundleID:     rec.BundleID,
			Size:         humanSize(rec.SizeBytes),
			InstalledAt:  timePtrString(rec.InstalledAt),
			DownloadURL:  rec.DownloadURL,
			ReinstallCmd: rec.ReinstallCmd,
			Notes:        strings.Join(rec.CollectorNotes, "; "),
		})
	}

	hostname, _ := os.Hostname()
	ctx := htmlContext{
		GeneratedAt: r.Now().Format(time.RFC3339),
		Hostname:    hostname,
		OS:          runtime.GOOS + "/" + runtime.GOARCH,
		Tool:        "osaat " + version.Version,
		Count:       len(rows),
		Records:     rows,
	}

	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, ctx)
}

const htmlTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>osaat — application inventory</title>
<style>
  :root {
    color-scheme: light dark;
    --border: #e2e2e2;
    --muted: #6b7280;
    --accent: #1d4ed8;
  }
  @media (prefers-color-scheme: dark) {
    :root { --border: #2a2a2a; --muted: #9ca3af; --accent: #93c5fd; }
  }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif; margin: 2rem; line-height: 1.45; }
  h1 { margin-top: 0; }
  .meta { color: var(--muted); font-size: 0.85rem; margin-bottom: 1.5rem; }
  .meta code { background: rgba(125,125,125,0.15); padding: 0.1rem 0.3rem; border-radius: 3px; }
  table { border-collapse: collapse; width: 100%; font-size: 0.875rem; }
  th, td { padding: 0.4rem 0.6rem; border-bottom: 1px solid var(--border); text-align: left; vertical-align: top; }
  th { cursor: pointer; user-select: none; position: sticky; top: 0; background: inherit; }
  th:hover { color: var(--accent); }
  td.num { text-align: right; font-variant-numeric: tabular-nums; }
  tr:hover { background: rgba(125,125,125,0.06); }
  .src-app_store { color: var(--accent); }
  .src-homebrew_cask, .src-homebrew_formula { color: #b45309; }
  .src-system { color: var(--muted); }
  .src-unknown { color: #dc2626; }
  .sig-unsigned, .sig-unknown { color: #dc2626; }
  .filter { margin-bottom: 1rem; }
  .filter input { padding: 0.4rem 0.6rem; font-size: 0.9rem; min-width: 18rem; }
</style>
</head>
<body>
<h1>Application inventory</h1>
<div class="meta">
  Generated <code>{{.GeneratedAt}}</code> on <code>{{.Hostname}}</code>
  ({{.OS}}) by <code>{{.Tool}}</code> — {{.Count}} applications.
</div>

<div class="filter">
  <input id="filter" type="search" placeholder="Filter by name, source, signing, author..." autofocus>
</div>

<table id="apps">
  <thead>
    <tr>
      <th data-sort="text">Name</th>
      <th data-sort="text">Version</th>
      <th data-sort="text">Source</th>
      <th data-sort="text">Signing</th>
      <th data-sort="text">Author</th>
      <th data-sort="num">Size</th>
      <th data-sort="text">Bundle ID</th>
      <th data-sort="text">Reinstall</th>
      <th data-sort="text">Notes</th>
    </tr>
  </thead>
  <tbody>
    {{range .Records}}
    <tr>
      <td>{{.Name}}</td>
      <td>{{.Version}}</td>
      <td class="src-{{.Source}}">{{.Source}}</td>
      <td class="sig-{{.Signing}}">{{.Signing}}{{if .SigningTeam}} <small>({{.SigningTeam}})</small>{{end}}</td>
      <td>{{.Author}}</td>
      <td class="num" data-value="{{.Size}}">{{.Size}}</td>
      <td><code>{{.BundleID}}</code></td>
      <td>{{if .ReinstallCmd}}<code>{{.ReinstallCmd}}</code>{{end}}</td>
      <td>{{.Notes}}</td>
    </tr>
    {{end}}
  </tbody>
</table>

<script>
(function () {
  var table = document.getElementById('apps');
  var tbody = table.tBodies[0];
  var headers = Array.from(table.querySelectorAll('th'));
  var filter = document.getElementById('filter');

  headers.forEach(function (th, idx) {
    th.addEventListener('click', function () {
      var rows = Array.from(tbody.rows);
      var sortType = th.dataset.sort || 'text';
      var asc = th.dataset.asc !== 'true';
      headers.forEach(function (h) { h.dataset.asc = ''; });
      th.dataset.asc = asc ? 'true' : 'false';
      rows.sort(function (a, b) {
        var av = a.cells[idx].innerText.trim();
        var bv = b.cells[idx].innerText.trim();
        if (sortType === 'num') {
          av = parseFloat(av) || 0;
          bv = parseFloat(bv) || 0;
        }
        if (av < bv) return asc ? -1 : 1;
        if (av > bv) return asc ? 1 : -1;
        return 0;
      });
      rows.forEach(function (r) { tbody.appendChild(r); });
    });
  });

  filter.addEventListener('input', function () {
    var q = filter.value.toLowerCase();
    Array.from(tbody.rows).forEach(function (row) {
      row.style.display = row.innerText.toLowerCase().indexOf(q) === -1 ? 'none' : '';
    });
  });
})();
</script>
</body>
</html>
`
