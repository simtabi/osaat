package wizard

import (
	"strings"
	"testing"
)

func TestReplayCommandIncludesAllSetFields(t *testing.T) {
	opts := Options{
		OS:              []string{"macos", "linux"},
		Formats:         []string{"pdf", "markdown"},
		Out:             "/Users/example/Documents/osaat/2026-05-16",
		LicenseMode:     "best-effort",
		AgeRecipient:    "age1qz8tnxw7p",
		Insights:        []string{"forgotten", "apple-silicon"},
		ForgottenMonths: 12,
		Restore:         true,
	}
	got := ReplayCommand(opts)
	for _, want := range []string{
		"osaat scan",
		"--os macos,linux",
		"--format pdf,markdown",
		"--out /Users/example/Documents/osaat/2026-05-16",
		"--license-mode best-effort",
		"--age-recipient age1qz8tnxw7p",
		"--insights forgotten,apple-silicon",
		"--insights-forgotten-months 12",
		"--with-restore",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("replay missing %q\ngot: %s", want, got)
		}
	}
}

func TestReplayCommandOmitsDefaults(t *testing.T) {
	opts := Options{
		Formats: []string{"json"},
		// LicenseMode == "none" is the default — should be omitted.
		LicenseMode:     "none",
		ForgottenMonths: 6, // also default
	}
	got := ReplayCommand(opts)
	for _, unwanted := range []string{"--license-mode", "--insights-forgotten-months"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("replay should not include %q (it's at default); got: %s", unwanted, got)
		}
	}
}

func TestReplayCommandQuotesPathsWithSpaces(t *testing.T) {
	opts := Options{Out: "/Users/test name/With Space"}
	got := ReplayCommand(opts)
	if !strings.Contains(got, "'/Users/test name/With Space'") {
		t.Errorf("path with spaces should be single-quoted; got: %s", got)
	}
}

func TestShellQuoteEscapesSingleQuote(t *testing.T) {
	if got := shellQuote("can't stop"); got != `'can'\''t stop'` {
		t.Errorf(`shellQuote escape wrong: got %s`, got)
	}
}
