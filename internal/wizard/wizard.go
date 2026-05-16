package wizard

import (
	"fmt"
	"runtime"

	"github.com/charmbracelet/huh"

	"github.com/simtabi/osaat/internal/paths"
)

// Run displays the interactive wizard and returns the populated
// Options. defaults seeds the form's initial values (typically from a
// --profile flag or the OS-aware defaults).
//
// The form is built from five groups:
//  1. Scope — which collectors to run.
//  2. Output — where to write outputs + which formats (and always
//     prompts the user for the output directory).
//  3. License — extraction mode + optional age recipient.
//  4. Insights — extra columns (forgotten apps, Apple Silicon).
//  5. Save — optional profile name to persist these answers.
func Run(defaults Options) (Options, error) {
	opts := defaults
	if len(opts.OS) == 0 {
		opts.OS = []string{DefaultOSFromRuntime()}
	}
	if len(opts.Formats) == 0 {
		opts.Formats = DefaultFormats()
	}
	if opts.Out == "" {
		if def, err := paths.DefaultOutputDir(); err == nil {
			opts.Out = def
		}
	}
	if opts.LicenseMode == "" {
		opts.LicenseMode = "none"
	}
	if opts.ForgottenMonths == 0 {
		opts.ForgottenMonths = 6
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Which collectors should run?").
				Description("Tick every OS you want to audit. Defaults to the OS you're on.").
				Options(
					huh.NewOption("macOS", "macos"),
					huh.NewOption("Linux (dpkg / rpm / pacman / snap / flatpak)", "linux"),
					huh.NewOption("Unix / BSD", "unix"),
				).
				Value(&opts.OS),
		).Title("Scope"),

		huh.NewGroup(
			huh.NewInput().
				Title("Where should generated files go?").
				Description("Default: Documents/osaat/<date>. Path will be created if missing.").
				Value(&opts.Out),
			huh.NewMultiSelect[string]().
				Title("Output formats").
				Description("Defaults: PDF, Markdown, plain text, JSON. Pick any combination.").
				Options(
					huh.NewOption("PDF (printable, well-formatted)", "pdf"),
					huh.NewOption("Markdown (.md, for sharing)", "markdown"),
					huh.NewOption("Plain text (.txt)", "txt"),
					huh.NewOption("JSON (machine-readable; required for osaat diff)", "json"),
					huh.NewOption("CSV (spreadsheet)", "csv"),
					huh.NewOption("HTML (browser, sortable)", "html"),
				).
				Value(&opts.Formats),
			huh.NewConfirm().
				Title("Also write the restoration manifest?").
				Description("Brewfile + mas-apps.txt + RESTORE.md").
				Value(&opts.Restore),
		).Title("Output"),

		huh.NewGroup(
			huh.NewSelect[string]().
				Title("License extraction mode").
				Description("Keys go to secrets.json (mode 600), never to report.json.").
				Options(
					huh.NewOption("None — skip license scan", "none"),
					huh.NewOption("Checklist — emit 'look here' pointers, no extraction", "checklist"),
					huh.NewOption("Best-effort — read plists, regex-match license fields", "best-effort"),
					huh.NewOption("Aggressive — best-effort plus a manual Keychain pointer", "aggressive"),
				).
				Value(&opts.LicenseMode),
			huh.NewInput().
				Title("Encrypt secrets.json with this age recipient (optional)").
				Description("Format: age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx — produces secrets.json.age instead of secrets.json. Leave blank to skip.").
				Value(&opts.AgeRecipient),
		).Title("License & encryption"),

		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Optional insight columns").
				Description("Land in Phase 3 — the wizard records your preference now.").
				Options(
					huh.NewOption("Forgotten apps (not opened in N months)", "forgotten"),
					huh.NewOption("Apple Silicon compatibility (macOS only)", "apple-silicon"),
				).
				Value(&opts.Insights),
		).Title("Insights"),

		huh.NewGroup(
			huh.NewInput().
				Title("Save these answers as a profile? (optional)").
				Description("Enter a name like 'imani-mbp' to save; leave blank to skip.").
				Value(&opts.SaveProfile),
		).Title("Save profile"),
	)

	if err := form.Run(); err != nil {
		return opts, fmt.Errorf("wizard: %w", err)
	}
	return opts, nil
}

// DefaultOSFromRuntime returns the wizard token for the host OS.
func DefaultOSFromRuntime() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	case "linux":
		return "linux"
	default:
		return "unix"
	}
}
