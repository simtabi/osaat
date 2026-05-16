// Package wizard implements the interactive `huh`-based form that
// gathers a complete osaat scan configuration, plus the bubbletea
// "live scan view" displayed while a scan runs.
//
// The wizard never invokes the collector directly. It produces an
// Options value that the CLI translates into the same flag set a
// non-interactive invocation would build. This keeps wizard mode and
// flag mode behaviorally identical.
package wizard

// Options is the wizard's output — a fully-populated scan
// configuration. The CLI consumes it as if every field had been
// passed on the command line.
type Options struct {
	OS              []string
	Formats         []string
	Out             string
	LicenseMode     string
	AgeRecipient    string
	Insights        []string
	ForgottenMonths int
	Restore         bool
	SaveProfile     string
}

// DefaultFormats is the wizard's pre-selected format set: PDF +
// Markdown + plain text, plus JSON (needed for osaat diff).
func DefaultFormats() []string {
	return []string{"pdf", "markdown", "txt", "json"}
}

// AllFormats lists every reporter the user can pick from in the
// wizard's multi-select.
func AllFormats() []string {
	return []string{"pdf", "markdown", "txt", "json", "csv", "html"}
}
