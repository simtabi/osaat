package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/simtabi/osaat/internal/diff"
)

func newDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <old.json> <new.json>",
		Short: "Diff two audit reports",
		Long: `Diff two audit reports. Records are matched on BundleID (macOS) or
PkgID (Linux/Unix), falling back to Name + Version when neither is set.

Exit codes:
  0  no differences
  1  differences found
  2  flag misuse or invalid input`,
		Args: cobra.ExactArgs(2),
		RunE: runDiff,
	}

	cmd.Flags().String("format", "text", "output format: text|json|markdown")

	return cmd
}

func runDiff(cmd *cobra.Command, args []string) error {
	oldPath, newPath := args[0], args[1]
	format, _ := cmd.Flags().GetString("format")

	oldRecs, err := diff.LoadReport(oldPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", oldPath, err)
	}
	newRecs, err := diff.LoadReport(newPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", newPath, err)
	}

	result := diff.Compare(oldRecs, newRecs)
	result.OldFile = oldPath
	result.NewFile = newPath

	w := cmd.OutOrStdout()
	switch format {
	case "text", "":
		if err := diff.WriteText(result, w); err != nil {
			return err
		}
	case "json":
		if err := diff.WriteJSON(result, w); err != nil {
			return err
		}
	case "markdown", "md":
		if err := diff.WriteMarkdown(result, w); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown --format: %s (use text|json|markdown)", format)
	}

	if !result.IsClean() {
		// Bypass cobra's error wrap so the exit code is exactly 1
		// without an extra "Error:" prefix on stderr. Per docs:
		//   0 = no diff, 1 = differences, 2 = misuse.
		os.Exit(1)
	}
	return nil
}
