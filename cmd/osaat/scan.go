package main

import "github.com/spf13/cobra"

func newScanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan the system and produce an audit report",
		Long: `Scan the system, produce an audit report and optionally a restoration manifest.

When stdin is a TTY and no flags are passed, scan opens an interactive
wizard. Pass --non-interactive to forbid the wizard, or --interactive to
force it.`,
		RunE: stubRun("scan"),
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
