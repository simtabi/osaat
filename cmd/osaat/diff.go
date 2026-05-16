package main

import "github.com/spf13/cobra"

func newDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <old.json> <new.json>",
		Short: "Diff two audit reports",
		Long: `Diff two audit reports. Records are matched on BundleID (macOS) or
PkgID (Linux/Unix), falling back to Name + Version when neither is set.`,
		Args: cobra.ExactArgs(2),
		RunE: stubRun("diff"),
	}

	cmd.Flags().String("format", "text", "output format: text|json|markdown")

	return cmd
}
