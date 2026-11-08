package main

import "github.com/spf13/cobra"

func newRestoreHelpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore-help",
		Short: "Emit a per-app manual-install checklist from a previously generated audit report",
		RunE:  stubRun("restore-help"),
	}

	cmd.Flags().String("from", "", "path to report.json (required)")
	cmd.Flags().String("output", "RESTORE.md", "path to write the checklist")
	_ = cmd.MarkFlagRequired("from")

	return cmd
}
