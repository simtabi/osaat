package wizard

import (
	"fmt"
	"regexp"
	"strings"
)

// ReplayCommand renders the equivalent `osaat scan` invocation for an
// Options value. The wizard prints this at the end so the user can
// see what their answers map to and re-run headlessly next time.
//
// The output is a single shell-safe line. Values that need quoting
// are wrapped in single quotes with embedded single quotes escaped.
func ReplayCommand(opts Options) string {
	var b strings.Builder
	b.WriteString("osaat scan")

	appendList := func(flag string, values []string) {
		if len(values) == 0 {
			return
		}
		fmt.Fprintf(&b, " %s %s", flag, strings.Join(values, ","))
	}
	appendString := func(flag, value string) {
		if value == "" {
			return
		}
		fmt.Fprintf(&b, " %s %s", flag, shellQuote(value))
	}

	appendList("--os", opts.OS)
	appendList("--format", opts.Formats)
	appendString("--out", opts.Out)
	if opts.LicenseMode != "" && opts.LicenseMode != "none" {
		fmt.Fprintf(&b, " --license-mode %s", opts.LicenseMode)
	}
	if opts.AgeRecipient != "" {
		fmt.Fprintf(&b, " --age-recipient %s", opts.AgeRecipient)
	}
	appendList("--insights", opts.Insights)
	if opts.ForgottenMonths != 0 && opts.ForgottenMonths != 6 {
		fmt.Fprintf(&b, " --insights-forgotten-months %d", opts.ForgottenMonths)
	}
	if opts.Restore {
		b.WriteString(" --with-restore")
	}
	return b.String()
}

// safeChars matches characters that never need shell quoting.
var safeChars = regexp.MustCompile(`^[A-Za-z0-9_./@:=+,-]+$`)

// shellQuote returns s wrapped in single quotes when it contains any
// character that would need escaping in a POSIX shell. Embedded single
// quotes are encoded via the standard '\'' trick.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if safeChars.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
