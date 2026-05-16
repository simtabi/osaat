package schedule

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCadenceValidate(t *testing.T) {
	for _, ok := range []Cadence{CadenceDaily, CadenceWeekly, CadenceMonthly} {
		if err := ok.Validate(); err != nil {
			t.Errorf("Validate(%q) returned %v", ok, err)
		}
	}
	for _, bad := range []Cadence{"", "yearly", "biweekly"} {
		if err := bad.Validate(); err == nil {
			t.Errorf("Validate(%q) should error", bad)
		}
	}
}

func TestLaunchdRenderWeekly(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	l := NewLaunchd()
	plan, err := l.Plan(InstallOptions{
		BinaryPath: "/usr/local/bin/osaat",
		Cadence:    CadenceWeekly,
		Profile:    "imani-mbp",
		DryRun:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) < 1 || plan.Actions[0].Kind != "write" {
		t.Fatalf("first action should be a write; got %+v", plan.Actions)
	}
	body := plan.Actions[0].Body
	for _, want := range []string{
		"<key>Label</key>",
		"<string>com.simtabi.osaat</string>",
		"<string>/usr/local/bin/osaat</string>",
		"<string>scan</string>",
		"<string>--profile</string>",
		"<string>imani-mbp</string>",
		"<key>Weekday</key>",
		"<integer>1</integer>",
		"<key>Hour</key>",
		"<integer>6</integer>",
		"launchd-stdout.log",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("launchd plist missing %q\n--- body ---\n%s", want, body)
		}
	}
	if plan.Actions[0].Path == "" || !strings.HasSuffix(plan.Actions[0].Path, ".plist") {
		t.Errorf("plist path looks wrong: %q", plan.Actions[0].Path)
	}
	if !strings.Contains(plan.Actions[0].Path, "LaunchAgents") {
		t.Errorf("plist path should be under LaunchAgents: %q", plan.Actions[0].Path)
	}
}

func TestLaunchdRenderDailyOmitsWeekdayDay(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	plan, err := NewLaunchd().Plan(InstallOptions{
		BinaryPath: "/x/osaat",
		Cadence:    CadenceDaily,
		DryRun:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	body := plan.Actions[0].Body
	if strings.Contains(body, "<key>Weekday</key>") {
		t.Errorf("daily plist should not include Weekday: %s", body)
	}
	if strings.Contains(body, "<key>Day</key>") {
		t.Errorf("daily plist should not include Day: %s", body)
	}
}

func TestLaunchdRenderMonthly(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	plan, err := NewLaunchd().Plan(InstallOptions{
		BinaryPath: "/x/osaat",
		Cadence:    CadenceMonthly,
		DryRun:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	body := plan.Actions[0].Body
	if !strings.Contains(body, "<key>Day</key>") {
		t.Errorf("monthly plist should include Day: %s", body)
	}
	if !strings.Contains(body, "<integer>1</integer>") {
		t.Errorf("monthly plist should target the 1st of the month: %s", body)
	}
}

func TestSystemdRenderWeekly(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	plan, err := NewSystemd().Plan(InstallOptions{
		BinaryPath: "/usr/local/bin/osaat",
		Cadence:    CadenceWeekly,
		Profile:    "imani-mbp",
		DryRun:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) < 3 {
		t.Fatalf("expected 3 actions (service + timer + load); got %d", len(plan.Actions))
	}

	var service, timer Action
	for _, a := range plan.Actions {
		if a.Kind != "write" {
			continue
		}
		switch {
		case strings.HasSuffix(a.Path, ".service"):
			service = a
		case strings.HasSuffix(a.Path, ".timer"):
			timer = a
		}
	}
	for _, want := range []string{
		"Type=oneshot",
		"ExecStart=/usr/local/bin/osaat scan --non-interactive",
		"--profile imani-mbp",
	} {
		if !strings.Contains(service.Body, want) {
			t.Errorf("service unit missing %q\n--- body ---\n%s", want, service.Body)
		}
	}
	for _, want := range []string{
		"[Timer]",
		"OnCalendar=Mon *-*-* 06:00:00",
		"WantedBy=timers.target",
		"Unit=osaat.service",
	} {
		if !strings.Contains(timer.Body, want) {
			t.Errorf("timer unit missing %q\n--- body ---\n%s", want, timer.Body)
		}
	}
}

func TestSystemdUninstallPlan(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	plan, err := NewSystemd().Uninstall(context.Background(), UninstallOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Actions[0].Kind != "unload" {
		t.Errorf("first action should be unload; got %+v", plan.Actions[0])
	}
	for _, a := range plan.Actions[1:] {
		if a.Kind != "remove" {
			t.Errorf("expected remove actions after unload; got %+v", a)
		}
	}
}

func TestForReturnsManagerForCurrentOS(t *testing.T) {
	m, err := For()
	if err != nil {
		// Acceptable on non-darwin/non-linux runners.
		t.Skipf("install-schedule not supported here: %v", err)
	}
	if m == nil {
		t.Fatal("For() returned nil manager")
	}
}

func TestInstallDryRunDoesNotWrite(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	mgr, err := For()
	if err != nil {
		t.Skip(err)
	}
	plan, err := mgr.Install(context.Background(), InstallOptions{
		BinaryPath: "/x/osaat",
		Cadence:    CadenceWeekly,
		DryRun:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.DryRun {
		t.Error("plan should be marked DryRun")
	}
	// Nothing should have been written to disk.
	for _, a := range plan.Actions {
		if a.Kind == "write" {
			if _, err := os.Stat(a.Path); err == nil {
				t.Errorf("dry run should not write %s", a.Path)
			}
		}
	}
	_ = filepath.SkipDir // keep filepath imported for any future helpers
}
