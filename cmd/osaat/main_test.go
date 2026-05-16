package main

import (
	"bytes"
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
// `scan` and `diff` ship in Phase 1 and Phase 2a respectively and are
// covered by dedicated tests.
func TestStubSubcommandsRun(t *testing.T) {
	for _, sub := range []string{"restore-help --from /tmp/x", "install-schedule", "backup"} {
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

// TestScanRejectsUnsupportedOS ensures scan exits cleanly when asked to
// run a collector that hasn't shipped yet. The check is fast (no real
// I/O) and guards the user-visible error message.
func TestScanRejectsUnsupportedOS(t *testing.T) {
	for _, osFlag := range []string{"linux", "unix"} {
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
