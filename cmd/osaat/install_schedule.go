package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/simtabi/osaat/internal/paths"
	"github.com/simtabi/osaat/internal/schedule"
)

func newInstallScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install-schedule",
		Short: "Install or remove a recurring scan via launchd (macOS) or systemd (Linux)",
		Long: `Install or remove a recurring osaat scan.

On macOS, writes a LaunchAgent plist to ~/Library/LaunchAgents/ and
loads it with launchctl. On Linux, writes a systemd user .service +
.timer pair to ~/.config/systemd/user/ and enables them with
systemctl --user.

Exactly one of --daily, --weekly, or --monthly must be supplied
unless --uninstall is set. The schedule always fires at 06:00 local
time (matches the rest of Simtabi's tooling).`,
		RunE: runInstallSchedule,
	}

	cmd.Flags().Bool("weekly", false, "schedule weekly (Mondays at 06:00)")
	cmd.Flags().Bool("daily", false, "schedule daily (06:00)")
	cmd.Flags().Bool("monthly", false, "schedule monthly (1st of month at 06:00)")
	cmd.Flags().Bool("uninstall", false, "remove the schedule")
	cmd.Flags().Bool("dry-run", false, "print what would be written; do nothing")
	cmd.Flags().String("label", "", "scheduler label (default 'com.simtabi.osaat' / 'osaat')")
	cmd.Flags().StringSlice("extra-arg", nil, "extra arguments appended to the scheduled scan (repeatable)")

	return cmd
}

func runInstallSchedule(cmd *cobra.Command, _ []string) error {
	uninstall, _ := cmd.Flags().GetBool("uninstall")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	label, _ := cmd.Flags().GetString("label")
	profile, _ := cmd.Flags().GetString("profile")
	outDir, _ := cmd.Flags().GetString("out")
	extra, _ := cmd.Flags().GetStringSlice("extra-arg")

	mgr, err := schedule.For()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if uninstall {
		plan, err := mgr.Uninstall(ctx, schedule.UninstallOptions{
			Label:  label,
			DryRun: dryRun,
		})
		printPlan(cmd, plan)
		return err
	}

	cadence, err := pickCadence(cmd)
	if err != nil {
		return err
	}

	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve osaat binary path: %w", err)
	}
	binPath, err = absExe(binPath)
	if err != nil {
		return err
	}

	if outDir == "" || outDir == "./osaat-out" {
		// For scheduled runs we want a sensible per-day default that
		// the scan binary will expand at execution time.
		if home, err := os.UserHomeDir(); err == nil {
			outDir = home + "/Documents/osaat/{date}"
		}
	}

	plan, err := mgr.Install(ctx, schedule.InstallOptions{
		BinaryPath: binPath,
		Cadence:    cadence,
		OutDir:     outDir,
		Profile:    profile,
		ExtraArgs:  extra,
		Label:      label,
		DryRun:     dryRun,
	})
	printPlan(cmd, plan)
	return err
}

func pickCadence(cmd *cobra.Command) (schedule.Cadence, error) {
	weekly, _ := cmd.Flags().GetBool("weekly")
	daily, _ := cmd.Flags().GetBool("daily")
	monthly, _ := cmd.Flags().GetBool("monthly")

	count := 0
	var c schedule.Cadence
	if weekly {
		c = schedule.CadenceWeekly
		count++
	}
	if daily {
		c = schedule.CadenceDaily
		count++
	}
	if monthly {
		c = schedule.CadenceMonthly
		count++
	}
	if count == 0 {
		return "", fmt.Errorf("one of --daily, --weekly, or --monthly is required (or pass --uninstall)")
	}
	if count > 1 {
		return "", fmt.Errorf("--daily / --weekly / --monthly are mutually exclusive")
	}
	return c, nil
}

func printPlan(cmd *cobra.Command, plan schedule.Plan) {
	w := cmd.OutOrStdout()
	if plan.DryRun {
		fmt.Fprintln(w, "Dry run — no changes made.")
	}
	for _, a := range plan.Actions {
		switch a.Kind {
		case "write":
			fmt.Fprintf(w, "write   %s  (mode %o, %d bytes)\n", paths.TidyPath(a.Path), a.Mode, len(a.Body))
		case "remove":
			fmt.Fprintf(w, "remove  %s\n", paths.TidyPath(a.Path))
		case "load":
			fmt.Fprintf(w, "load    %s\n", a.Path)
		case "unload":
			fmt.Fprintf(w, "unload  %s\n", a.Path)
		default:
			fmt.Fprintf(w, "%-7s %s\n", a.Kind, a.Path)
		}
	}
}

// absExe returns the absolute path of the running binary, resolving
// any symlinks. The scheduler needs an absolute, stable path —
// relative paths like "./bin/osaat" won't work after a reboot.
func absExe(p string) (string, error) {
	abs, err := osReadlink(p)
	if err != nil {
		return p, nil
	}
	if abs == "" {
		return p, nil
	}
	return abs, nil
}

// osReadlink is a thin wrapper around os.Readlink that returns the
// input path when it isn't a symlink. Kept separate so the install
// path can be unit-tested without filesystem manipulation later.
func osReadlink(p string) (string, error) {
	info, err := os.Lstat(p)
	if err != nil {
		return p, nil
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return p, nil
	}
	return os.Readlink(p)
}
