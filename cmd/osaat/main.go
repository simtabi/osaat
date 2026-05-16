// Command osaat is the entrypoint for the osaat CLI.
// Subcommands are defined in sibling files.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "osaat",
		Short: "Audit and back up installed applications on macOS, Linux, and Unix",
		Long: `osaat audits installed applications on a macOS, Linux, or Unix machine
and emits a structured inventory plus a restoration manifest you can run
on a new machine.

When stdin is a TTY and no flags are passed, ` + "`scan`" + ` opens an
interactive wizard. Otherwise it runs headlessly with the flag values.`,
		SilenceUsage: true,
	}

	root.PersistentFlags().String("out", "./osaat-out", "output directory")
	root.PersistentFlags().String("profile", "", "named profile under ~/.config/osaat/profiles/")
	root.PersistentFlags().Bool("non-interactive", false, "disable interactive wizard")
	root.PersistentFlags().Bool("verbose", false, "verbose logging")
	root.PersistentFlags().Bool("quiet", false, "suppress per-file `wrote ...` lines")

	root.AddCommand(newScanCmd())
	root.AddCommand(newDiffCmd())
	root.AddCommand(newRestoreHelpCmd())
	root.AddCommand(newInstallScheduleCmd())
	root.AddCommand(newBackupCmd())
	root.AddCommand(newVersionCmd())

	return root
}

// stubRun returns a RunE that prints a not-implemented notice. It is
// used by every subcommand during Phase 0; each subcommand replaces its
// RunE with real logic in a later phase.
func stubRun(name string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		fmt.Fprintf(cmd.OutOrStdout(), "[%s] not implemented yet — see .design/plans/ for the build plan\n", name)
		return nil
	}
}
