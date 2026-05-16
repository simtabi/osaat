package main

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
)

func TestRootHelpListsAllSubcommands(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("--help returned error: %v", err)
	}

	want := []string{
		"scan",
		"diff",
		"restore-help",
		"install-schedule",
		"backup",
		"version",
	}
	out := buf.String()
	for _, sub := range want {
		if !strings.Contains(out, sub) {
			t.Errorf("--help output missing subcommand %q\n--- output ---\n%s", sub, out)
		}
	}
}

func TestVersionPrintsDefault(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("version returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "0.0.0-dev") {
		t.Errorf("expected default version 0.0.0-dev in output:\n%s", buf.String())
	}
}

// TestStubSubcommandsRun exercises subcommands that are still stubs.
// scan, diff, install-schedule, and backup ship in earlier phases and
// are covered by dedicated tests.
func TestStubSubcommandsRun(t *testing.T) {
	for _, sub := range []string{"restore-help --from /tmp/x"} {
		t.Run(sub, func(t *testing.T) {
			cmd := newRootCmd()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(strings.Fields(sub))
			if err := cmd.Execute(); err != nil {
				t.Fatalf("%s returned error: %v", sub, err)
			}
			if !strings.Contains(buf.String(), "not implemented yet") {
				t.Errorf("expected stub notice for %s; got:\n%s", sub, buf.String())
			}
		})
	}
}

// TestInstallScheduleDryRun confirms the scheduler subcommand renders
// a plan without touching disk when --dry-run is passed.
func TestInstallScheduleDryRun(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"install-schedule", "--weekly", "--dry-run"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("install-schedule --weekly --dry-run failed: %v", err)
	}
	for _, want := range []string{"Dry run", "write"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("dry-run output missing %q\n%s", want, buf.String())
		}
	}
}

// TestBackupRequiresArguments ensures the backup subcommand surfaces
// clean errors when invoked without the necessary flags. Both modes
// short-circuit before any I/O.
func TestBackupRequiresArguments(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"create missing --from", []string{"backup", "--age-recipient", "age1xyz"}, "--from"},
		{"create missing --age-recipient", []string{"backup", "--from", "/tmp"}, "--age-recipient"},
		{"decrypt missing --in", []string{"backup", "--decrypt"}, "--in"},
		{"decrypt missing --out", []string{"backup", "--decrypt", "--in", "/tmp/x.tar.age"}, "--out"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCmd()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error for %v", tc.args)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error should mention %q; got: %v", tc.want, err)
			}
		})
	}
}

// TestScanRejectsUnsupportedOS ensures scan exits cleanly when asked to
// run a collector that hasn't shipped yet. Phase 4a added Linux, so
// only `unix` is checked here.
func TestScanRejectsUnsupportedOS(t *testing.T) {
	for _, osFlag := range []string{"unix"} {
		t.Run(osFlag, func(t *testing.T) {
			cmd := newRootCmd()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs([]string{"scan", "--os", osFlag, "--non-interactive"})
			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error for --os %s; got nil", osFlag)
			}
			if !strings.Contains(err.Error(), "not implemented") {
				t.Errorf("error for --os %s should mention not implemented; got: %v", osFlag, err)
			}
		})
	}
}

// TestScanLinuxOnNonLinuxErrors confirms the Linux collector returns
// a clean error message when invoked off-platform (this dev mac is
// darwin). Real Linux behavior is exercised via the parser unit tests
// and the CI matrix.
func TestScanLinuxOnNonLinuxErrors(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("running on Linux; this path tests the off-platform guard")
	}
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"scan", "--os", "linux", "--non-interactive", "--format", "json", "--out", t.TempDir()})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when running --os linux on non-Linux host")
	}
	if !strings.Contains(err.Error(), "linux collector requires linux") {
		t.Errorf("unexpected error: %v", err)
	}
}
