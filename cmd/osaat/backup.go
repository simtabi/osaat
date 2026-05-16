package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/simtabi/osaat/internal/paths"
	"github.com/simtabi/osaat/internal/restore"
)

func newBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Create or decrypt an encrypted backup archive (tar.age)",
		Long: `Bundle a scan output directory into a single age-encrypted tar
archive, or decrypt and extract one back to a directory.

Create mode (default):
  osaat backup --from <dir> --age-recipient age1xxx... [flags]

Decrypt mode:
  osaat backup --decrypt --in <file.tar.age> --age-key <path> [flags]

The archive includes the known osaat output set by default
(report.* / Brewfile / mas-apps.txt / RESTORE.md / SHA256SUMS /
secrets.json[.age] / osaat-metadata.json). Pass --include-extras to
copy every regular file in --from.`,
		RunE: runBackup,
	}

	cmd.Flags().String("from", "", "source directory (create mode)")
	cmd.Flags().String("age-recipient", "", "age recipient public key (create mode)")
	cmd.Flags().Bool("decrypt", false, "switch to decrypt mode")
	cmd.Flags().String("in", "", "encrypted archive path (decrypt mode)")
	cmd.Flags().String("age-key", "", "age private key (decrypt mode; default: ~/.age/key.txt if present)")
	cmd.Flags().Bool("include-extras", false, "include every regular file under --from, not just the known set")

	return cmd
}

func runBackup(cmd *cobra.Command, _ []string) error {
	decrypt, _ := cmd.Flags().GetBool("decrypt")
	if decrypt {
		return runBackupDecrypt(cmd)
	}
	return runBackupCreate(cmd)
}

func runBackupCreate(cmd *cobra.Command, _ ...string) error {
	from, _ := cmd.Flags().GetString("from")
	recipient, _ := cmd.Flags().GetString("age-recipient")
	out, _ := cmd.Flags().GetString("out")
	includeExtras, _ := cmd.Flags().GetBool("include-extras")
	quiet, _ := cmd.Flags().GetBool("quiet")

	if from == "" {
		return errors.New("--from <dir> is required in create mode")
	}
	if recipient == "" {
		return errors.New("--age-recipient is required in create mode")
	}
	if out == "" || out == "./osaat-out" {
		out = filepath.Join(filepath.Dir(from), filepath.Base(from)+".tar.age")
	} else if isDir(out) {
		out = filepath.Join(out, filepath.Base(from)+".tar.age")
	}

	recipients, err := restore.ParseRecipients([]string{recipient})
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(out), err)
	}
	f, err := os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create %s: %w", out, err)
	}
	defer f.Close()

	if err := restore.WriteArchive(f, restore.ArchiveOptions{
		SourceDir:     from,
		Recipients:    recipients,
		IncludeExtras: includeExtras,
	}); err != nil {
		return err
	}

	if !quiet {
		fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", paths.TidyPath(out))
	}
	return nil
}

func runBackupDecrypt(cmd *cobra.Command) error {
	in, _ := cmd.Flags().GetString("in")
	out, _ := cmd.Flags().GetString("out")
	ageKey, _ := cmd.Flags().GetString("age-key")
	quiet, _ := cmd.Flags().GetBool("quiet")

	if in == "" {
		return errors.New("--in <file.tar.age> is required in decrypt mode")
	}
	if out == "" || out == "./osaat-out" {
		return errors.New("--out <dir> is required in decrypt mode (use a fresh directory)")
	}

	keyPath, err := resolveAgeKey(ageKey)
	if err != nil {
		return err
	}
	identities, err := restore.LoadIdentitiesFromFile(keyPath)
	if err != nil {
		return err
	}

	f, err := os.Open(in)
	if err != nil {
		return fmt.Errorf("open %s: %w", in, err)
	}
	defer f.Close()

	written, err := restore.DecryptArchive(f, out, identities)
	if err != nil {
		return err
	}
	if !quiet {
		for _, p := range written {
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", paths.TidyPath(p))
		}
	}
	return nil
}

// runBackupCreate's variadic argument is unused; kept on the function
// signature to avoid breaking a future codepath that might want to
// thread extra arguments through.
var _ = runBackupCreate

// resolveAgeKey returns the supplied path, or falls back to
// $HOME/.age/key.txt when blank.
func resolveAgeKey(supplied string) (string, error) {
	if supplied != "" {
		return supplied, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	def := filepath.Join(home, ".age", "key.txt")
	if _, err := os.Stat(def); err != nil {
		return "", fmt.Errorf("--age-key is required (no default at %s)", paths.TidyPath(def))
	}
	return def, nil
}

// isDir reports whether p exists and is a directory.
func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
