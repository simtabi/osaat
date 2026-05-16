package main

import "github.com/spf13/cobra"

func newInstallScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install-schedule",
		Short: "Install or remove a recurring scan via launchd (macOS) or systemd (Linux)",
		RunE:  stubRun("install-schedule"),
	}

	cmd.Flags().Bool("weekly", false, "schedule weekly (Monday)")
	cmd.Flags().Bool("daily", false, "schedule daily")
	cmd.Flags().Bool("monthly", false, "schedule monthly (1st of month)")
	cmd.Flags().Bool("uninstall", false, "remove the schedule")
	cmd.Flags().Bool("dry-run", false, "print what would be written; do nothing")

	return cmd
}
