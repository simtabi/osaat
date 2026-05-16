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

func TestStubSubcommandsRun(t *testing.T) {
	for _, sub := range []string{"scan", "diff a b", "restore-help --from /tmp/x", "install-schedule", "backup"} {
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
