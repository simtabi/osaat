package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/simtabi/osaat/internal/collectors"
	"github.com/simtabi/osaat/internal/collectors/macos"
	"github.com/simtabi/osaat/internal/licenses"
	"github.com/simtabi/osaat/internal/reporters"
	"github.com/simtabi/osaat/internal/restore"
	"github.com/simtabi/osaat/internal/secrets"
)

func newScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan the system and produce an audit report",
		Long: `Scan the system, produce an audit report and optionally a restoration manifest.

When stdin is a TTY and no flags are passed, scan opens an interactive
wizard. Pass --non-interactive to forbid the wizard, or --interactive to
force it.`,
		RunE: runScan,
	}

	cmd.Flags().String("os", "auto", "collector set: macos|linux|unix|auto")
	cmd.Flags().StringSlice("format", []string{"json"}, "output formats: json,csv,markdown,html")
	cmd.Flags().String("license-mode", "none", "license extraction: none|best-effort|checklist|aggressive")
	cmd.Flags().String("age-recipient", "", "age recipient public key — encrypts secrets.json")
	cmd.Flags().StringSlice("insights", nil, "extra columns: forgotten,apple-silicon")
	cmd.Flags().Int("insights-forgotten-months", 6, "months of inactivity that flag an app as forgotten")
	cmd.Flags().Bool("with-restore", false, "also emit Brewfile, mas list, RESTORE.md")
	cmd.Flags().Bool("interactive", false, "force the interactive wizard")

	return cmd
}

func runScan(cmd *cobra.Command, _ []string) error {
	osFlag, _ := cmd.Flags().GetString("os")
	formats, _ := cmd.Flags().GetStringSlice("format")
	outDir, _ := cmd.Flags().GetString("out")
	verbose, _ := cmd.Flags().GetBool("verbose")
	licenseMode, _ := cmd.Flags().GetString("license-mode")

	if osFlag == "auto" {
		osFlag = runtime.GOOS
		if osFlag == "darwin" {
			osFlag = "macos"
		}
	}

	log := newLogger(verbose)

	scanner, err := licenses.For(licenseMode)
	if err != nil {
		return err
	}

	ageRecipient, _ := cmd.Flags().GetString("age-recipient")

	collector, err := collectorFor(osFlag, log)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	start := time.Now()
	log.Info("starting scan", "os", osFlag, "out", outDir)

	records, err := collector.Collect(ctx)
	if err != nil {
		return fmt.Errorf("collect: %w", err)
	}
	log.Info("scan complete", "duration", time.Since(start).Round(time.Millisecond), "records", len(records))

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create out dir: %w", err)
	}

	for _, raw := range formats {
		format := strings.TrimSpace(strings.ToLower(raw))
		rep, err := reporterFor(format)
		if err != nil {
			return err
		}
		ext := format
		if ext == "markdown" {
			ext = "md"
		}
		outPath := filepath.Join(outDir, "report."+ext)
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("create %s: %w", outPath, err)
		}
		if err := rep.Write(records, f); err != nil {
			_ = f.Close()
			return fmt.Errorf("write %s: %w", outPath, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("close %s: %w", outPath, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", outPath)
	}

	withRestore, _ := cmd.Flags().GetBool("with-restore")
	if withRestore {
		paths, err := restore.WriteAll(records, outDir)
		if err != nil {
			return fmt.Errorf("restore manifest: %w", err)
		}
		for _, p := range paths {
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", p)
		}
	}

	if scanner != nil {
		sec, err := scanner.Scan(ctx, records)
		if err != nil {
			return fmt.Errorf("license scan: %w", err)
		}
		if sec != nil && !sec.IsEmpty() {
			path, err := writeSecrets(sec, outDir, ageRecipient)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", path)
		} else {
			log.Info("license scan produced no findings; nothing to write")
		}
	}

	return nil
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

func newLogger(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
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
	default:
		return nil, fmt.Errorf("unknown --format: %s", format)
	}
}
