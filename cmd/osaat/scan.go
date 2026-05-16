package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/simtabi/osaat/internal/audit"
	"github.com/simtabi/osaat/internal/collectors"
	"github.com/simtabi/osaat/internal/collectors/macos"
	"github.com/simtabi/osaat/internal/licenses"
	"github.com/simtabi/osaat/internal/logging"
	"github.com/simtabi/osaat/internal/paths"
	"github.com/simtabi/osaat/internal/profiles"
	"github.com/simtabi/osaat/internal/reporters"
	"github.com/simtabi/osaat/internal/restore"
	"github.com/simtabi/osaat/internal/secrets"
	"github.com/simtabi/osaat/internal/wizard"
)

func newScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan the system and produce an audit report",
		Long: `Scan the system, produce an audit report and optionally a restoration manifest.

When stdin is a TTY and no scan flags are passed, scan opens an
interactive wizard built on the charmbracelet/huh form library. Pass
--non-interactive to forbid the wizard, or --interactive to force it.

Defaults follow the project conventions:
  - Output:   Documents/osaat/<YYYY-MM-DD>/ on every OS.
  - Logs:     $HOME/.config/osaat/logs/osaat-<YYYY-MM-DD>.log (mode 600).
  - Profiles: $HOME/.config/osaat/profiles/<name>.toml (mode 600).
  - Secrets:  secrets.json (mode 600) in the output directory, or
              secrets.json.age when --age-recipient is set.`,
		RunE: runScan,
	}

	cmd.Flags().String("os", "auto", "collector set: macos|linux|unix|auto")
	cmd.Flags().StringSlice("format", []string{"pdf", "markdown", "txt", "json"}, "output formats: pdf,markdown,txt,json,csv,html")
	cmd.Flags().String("license-mode", "none", "license extraction: none|best-effort|checklist|aggressive")
	cmd.Flags().String("age-recipient", "", "age recipient public key — encrypts secrets.json")
	cmd.Flags().StringSlice("insights", nil, "extra columns: forgotten,apple-silicon")
	cmd.Flags().Int("insights-forgotten-months", 6, "months of inactivity that flag an app as forgotten")
	cmd.Flags().Bool("with-restore", false, "also emit Brewfile, mas list, RESTORE.md")
	cmd.Flags().Bool("interactive", false, "force the interactive wizard")
	cmd.Flags().Bool("quiet", false, "suppress per-file `wrote ...` lines")

	return cmd
}

// scanInputs is the resolved configuration for one scan run, after
// flag parsing, profile loading, and (optionally) the interactive
// wizard have all had a chance to set values.
type scanInputs struct {
	OS              string
	Formats         []string
	Out             string
	LicenseMode     string
	AgeRecipient    string
	Insights        []string
	ForgottenMonths int
	WithRestore     bool
	SaveProfile     string

	Verbose     bool
	Quiet       bool
	Interactive bool
}

func runScan(cmd *cobra.Command, _ []string) error {
	in, err := resolveScanInputs(cmd)
	if err != nil {
		return err
	}

	// File logger: always on, always to $HOME/.config/osaat/logs/.
	// Stderr logger: tee'd in for non-interactive runs so the user
	// sees scan progress in their terminal.
	logger, closeLog, err := setupLogger(in.Verbose, in.Interactive)
	if err != nil {
		return err
	}
	defer closeLog()

	collector, err := collectorFor(in.OS, logger)
	if err != nil {
		return err
	}
	scanner, err := licenses.For(in.LicenseMode)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	start := time.Now()
	logger.Info("starting scan", "os", in.OS, "out", in.Out)

	if !in.Quiet {
		fmt.Fprintf(cmd.OutOrStdout(), "Scanning... outputs will land in %s\n", paths.TidyPath(in.Out))
	}

	records, err := collector.Collect(ctx)
	if err != nil {
		return fmt.Errorf("collect: %w", err)
	}
	duration := time.Since(start).Round(time.Millisecond)
	logger.Info("scan complete", "duration", duration, "records", len(records))

	if err := os.MkdirAll(in.Out, 0o755); err != nil {
		return fmt.Errorf("create out dir: %w", err)
	}

	written, err := writeReports(cmd, in, records)
	if err != nil {
		return err
	}

	if in.WithRestore {
		paths, err := restore.WriteAll(records, in.Out)
		if err != nil {
			return fmt.Errorf("restore manifest: %w", err)
		}
		for _, p := range paths {
			written = append(written, p)
			if !in.Quiet {
				fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", p)
			}
		}
	}

	if scanner != nil {
		sec, err := scanner.Scan(ctx, records)
		if err != nil {
			return fmt.Errorf("license scan: %w", err)
		}
		if sec != nil && !sec.IsEmpty() {
			path, err := writeSecrets(sec, in.Out, in.AgeRecipient)
			if err != nil {
				return err
			}
			written = append(written, path)
			if !in.Quiet {
				fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", path)
			}
		} else {
			logger.Info("license scan produced no findings; nothing to write")
		}
	}

	// SHA256 of every output for verifiable restore.
	checksumPath, err := writeChecksums(in.Out, written)
	if err != nil {
		logger.Warn("could not write checksums file", "err", err)
	} else if !in.Quiet {
		fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", checksumPath)
	}

	// Optional profile save (always when wizard set SaveProfile).
	if in.SaveProfile != "" {
		path, err := profiles.Save(in.SaveProfile, profileFromInputs(in))
		if err != nil {
			logger.Warn("could not save profile", "name", in.SaveProfile, "err", err)
		} else if !in.Quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "saved profile to %s\n", paths.TidyPath(path))
		}
	}

	if in.Interactive && !in.Quiet {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "Run this scan headlessly next time:")
		fmt.Fprintln(cmd.OutOrStdout(), "  "+wizard.ReplayCommand(optionsFromInputs(in)))
	}

	return nil
}

// resolveScanInputs collects values from flags, optionally a named
// profile, and optionally the interactive wizard.
func resolveScanInputs(cmd *cobra.Command) (scanInputs, error) {
	in := scanInputs{
		Verbose: getBool(cmd, "verbose"),
		Quiet:   getBool(cmd, "quiet"),
	}

	// Step 1: seed from flags. We re-read each flag below to know what
	// was explicitly set vs. left at default.
	in.OS, _ = cmd.Flags().GetString("os")
	in.Formats, _ = cmd.Flags().GetStringSlice("format")
	in.Out, _ = cmd.Flags().GetString("out")
	in.LicenseMode, _ = cmd.Flags().GetString("license-mode")
	in.AgeRecipient, _ = cmd.Flags().GetString("age-recipient")
	in.Insights, _ = cmd.Flags().GetStringSlice("insights")
	in.ForgottenMonths, _ = cmd.Flags().GetInt("insights-forgotten-months")
	in.WithRestore, _ = cmd.Flags().GetBool("with-restore")

	// Step 2: apply profile defaults, if requested. Profile values
	// only override flags the user did NOT explicitly set.
	profileName, _ := cmd.Flags().GetString("profile")
	if profileName != "" {
		p, err := profiles.Load(profileName)
		if err != nil {
			return in, fmt.Errorf("load profile: %w", err)
		}
		applyProfile(cmd, &in, p)
	}

	// Step 3: --os auto → host OS.
	if in.OS == "auto" {
		in.OS = runtime.GOOS
		if in.OS == "darwin" {
			in.OS = "macos"
		}
	}

	// Step 4: default output dir if still unset or at cobra's default.
	if in.Out == "" || in.Out == "./osaat-out" {
		if dir, err := paths.DefaultOutputDir(); err == nil {
			in.Out = dir
		}
	}

	// Step 5: decide whether to open the wizard.
	in.Interactive = shouldRunWizard(cmd)

	if in.Interactive {
		opts, err := wizard.Run(optionsFromInputs(in))
		if err != nil {
			return in, err
		}
		applyOptions(&in, opts)
	}

	return in, nil
}

// shouldRunWizard returns true when stdin is a TTY, --non-interactive
// was not set, and either --interactive was passed or no scan-shaping
// flags were given on the CLI.
func shouldRunWizard(cmd *cobra.Command) bool {
	nonInteractive := getBool(cmd, "non-interactive")
	interactive := getBool(cmd, "interactive")
	if nonInteractive {
		return false
	}
	if interactive {
		return true
	}
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		return false
	}
	// "No scan flags" check — if any of these were explicitly set on
	// the command line, the user is driving headlessly and the
	// wizard should stay out of the way.
	for _, name := range []string{"os", "format", "license-mode", "age-recipient", "insights", "with-restore", "out", "profile"} {
		if f := cmd.Flag(name); f != nil && f.Changed {
			return false
		}
	}
	return true
}

func setupLogger(verbose, interactive bool) (*slog.Logger, func(), error) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	logDir, err := paths.LogsDir()
	if err != nil {
		return nil, func() {}, err
	}
	fileLogger, fileCloser, err := logging.NewFileLogger(logDir, level)
	if err != nil {
		return nil, func() {}, err
	}

	// In wizard mode the terminal is owned by huh / bubbletea, so we
	// don't tee progress to stderr — it would corrupt the form. The
	// file logger captures everything regardless.
	if interactive {
		return fileLogger, func() { _ = fileCloser.Close() }, nil
	}

	stderrHandler := logging.NewScrubHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	logger := logging.NewTeeLogger(stderrHandler, fileLogger.Handler())
	return logger, func() { _ = fileCloser.Close() }, nil
}

func writeReports(cmd *cobra.Command, in scanInputs, records []audit.AppRecord) ([]string, error) {
	var written []string
	for _, raw := range in.Formats {
		format := strings.TrimSpace(strings.ToLower(raw))
		rep, err := reporterFor(format)
		if err != nil {
			return written, err
		}
		ext := format
		switch ext {
		case "markdown":
			ext = "md"
		case "text":
			ext = "txt"
		}
		outPath := filepath.Join(in.Out, "report."+ext)
		f, err := os.Create(outPath)
		if err != nil {
			return written, fmt.Errorf("create %s: %w", outPath, err)
		}
		if err := rep.Write(records, f); err != nil {
			_ = f.Close()
			return written, fmt.Errorf("write %s: %w", outPath, err)
		}
		if err := f.Close(); err != nil {
			return written, fmt.Errorf("close %s: %w", outPath, err)
		}
		written = append(written, outPath)
		if !in.Quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", outPath)
		}
	}
	return written, nil
}

// writeSecrets writes the secrets file to outDir, encrypted to the
// provided age recipient when set. Returns the path that was written.
// The plain file is chmod 600; the encrypted one is 644 (it's safe to
// share an age ciphertext).
func writeSecrets(sec *secrets.File, outDir, recipient string) (string, error) {
	if recipient == "" {
		path := filepath.Join(outDir, "secrets.json")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			return "", fmt.Errorf("create %s: %w", path, err)
		}
		defer f.Close()
		if err := secrets.WriteJSON(sec, f); err != nil {
			return "", fmt.Errorf("write %s: %w", path, err)
		}
		return path, nil
	}

	path := filepath.Join(outDir, "secrets.json.age")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	if err := secrets.WriteEncrypted(sec, []string{recipient}, f); err != nil {
		return "", fmt.Errorf("encrypt %s: %w", path, err)
	}
	return path, nil
}

// writeChecksums emits a sha256sum-compatible file at
// <outDir>/SHA256SUMS containing the digest of every written file.
func writeChecksums(outDir string, files []string) (string, error) {
	checksumPath := filepath.Join(outDir, "SHA256SUMS")
	out, err := os.Create(checksumPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	for _, p := range files {
		digest, err := fileSHA256(p)
		if err != nil {
			return checksumPath, err
		}
		base := filepath.Base(p)
		if _, err := fmt.Fprintf(out, "%s  %s\n", digest, base); err != nil {
			return checksumPath, err
		}
	}
	return checksumPath, nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func collectorFor(osFlag string, log *slog.Logger) (collectors.Collector, error) {
	switch osFlag {
	case "macos":
		return macos.New(macos.WithLogger(log)), nil
	case "linux":
		return nil, fmt.Errorf("--os linux is not implemented yet (Phase 4)")
	case "unix":
		return nil, fmt.Errorf("--os unix is not implemented yet (Phase 4)")
	default:
		return nil, fmt.Errorf("unsupported --os value: %s (use macos|linux|unix|auto)", osFlag)
	}
}

func reporterFor(format string) (reporters.Reporter, error) {
	switch format {
	case "json":
		return reporters.NewJSONReporter(), nil
	case "csv":
		return reporters.NewCSVReporter(), nil
	case "markdown", "md":
		return reporters.NewMarkdownReporter(), nil
	case "html":
		return reporters.NewHTMLReporter(), nil
	case "txt", "text":
		return reporters.NewTextReporter(), nil
	case "pdf":
		return reporters.NewPDFReporter(), nil
	default:
		return nil, fmt.Errorf("unknown --format: %s", format)
	}
}

func getBool(cmd *cobra.Command, name string) bool {
	if f := cmd.Flag(name); f == nil {
		return false
	}
	v, _ := cmd.Flags().GetBool(name)
	return v
}

func applyProfile(cmd *cobra.Command, in *scanInputs, p profiles.Profile) {
	changed := func(name string) bool {
		f := cmd.Flag(name)
		return f != nil && f.Changed
	}
	if !changed("os") && len(p.OS) > 0 {
		in.OS = p.OS[0]
	}
	if !changed("format") && len(p.Formats) > 0 {
		in.Formats = p.Formats
	}
	if !changed("out") && p.Out != "" {
		in.Out = expandOutTemplate(p.Out)
	}
	if !changed("license-mode") && p.License.Mode != "" {
		in.LicenseMode = p.License.Mode
	}
	if !changed("age-recipient") && p.License.AgeRecipient != "" {
		in.AgeRecipient = p.License.AgeRecipient
	}
	if !changed("insights") {
		if p.Insights.Forgotten {
			in.Insights = append(in.Insights, "forgotten")
		}
		if p.Insights.AppleSilicon {
			in.Insights = append(in.Insights, "apple-silicon")
		}
	}
	if !changed("insights-forgotten-months") && p.Insights.ForgottenMonths > 0 {
		in.ForgottenMonths = p.Insights.ForgottenMonths
	}
	if !changed("with-restore") {
		in.WithRestore = p.Restore.Enabled
	}
}

// expandOutTemplate substitutes {date} in profile-out paths with the
// current date.
func expandOutTemplate(in string) string {
	if !strings.Contains(in, "{date}") {
		return in
	}
	return strings.ReplaceAll(in, "{date}", time.Now().Format("2006-01-02"))
}

func optionsFromInputs(in scanInputs) wizard.Options {
	return wizard.Options{
		OS:              []string{in.OS},
		Formats:         append([]string{}, in.Formats...),
		Out:             in.Out,
		LicenseMode:     in.LicenseMode,
		AgeRecipient:    in.AgeRecipient,
		Insights:        append([]string{}, in.Insights...),
		ForgottenMonths: in.ForgottenMonths,
		Restore:         in.WithRestore,
		SaveProfile:     in.SaveProfile,
	}
}

func applyOptions(in *scanInputs, opts wizard.Options) {
	if len(opts.OS) > 0 {
		in.OS = opts.OS[0]
	}
	if len(opts.Formats) > 0 {
		in.Formats = opts.Formats
	}
	if opts.Out != "" {
		in.Out = opts.Out
	}
	if opts.LicenseMode != "" {
		in.LicenseMode = opts.LicenseMode
	}
	in.AgeRecipient = opts.AgeRecipient
	in.Insights = opts.Insights
	if opts.ForgottenMonths > 0 {
		in.ForgottenMonths = opts.ForgottenMonths
	}
	in.WithRestore = opts.Restore
	in.SaveProfile = opts.SaveProfile
}

func profileFromInputs(in scanInputs) profiles.Profile {
	p := profiles.Profile{
		OS:      []string{in.OS},
		Formats: append([]string{}, in.Formats...),
		Out:     in.Out,
		License: profiles.License{
			Mode:         in.LicenseMode,
			AgeRecipient: in.AgeRecipient,
		},
		Insights: profiles.Insights{
			ForgottenMonths: in.ForgottenMonths,
		},
		Restore: profiles.Restore{Enabled: in.WithRestore},
	}
	for _, ins := range in.Insights {
		switch ins {
		case "forgotten":
			p.Insights.Forgotten = true
		case "apple-silicon":
			p.Insights.AppleSilicon = true
		}
	}
	return p
}
