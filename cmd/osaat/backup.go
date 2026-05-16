package main

import "github.com/spf13/cobra"

func newBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create or decrypt an encrypted backup archive (tar.age)",
		RunE:  stubRun("backup"),
	}

	cmd.Flags().String("from", "", "source directory (create mode)")
	cmd.Flags().String("age-recipient", "", "age recipient public key (create mode)")
	cmd.Flags().Bool("decrypt", false, "switch to decrypt mode")
	cmd.Flags().String("in", "", "encrypted archive path (decrypt mode)")
	cmd.Flags().String("age-key", "", "age private key (decrypt mode)")
	cmd.Flags().Bool("include-extras", false, "include all files under --from, not just the known set")

	return cmd
}
