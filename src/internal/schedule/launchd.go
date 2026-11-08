package schedule

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/simtabi/osaat/internal/collectors"
)

// Launchd implements Manager for macOS via launchd LaunchAgents.
type Launchd struct{}

// NewLaunchd returns a launchd Manager.
func NewLaunchd() *Launchd { return &Launchd{} }

const launchdTemplateText = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinaryPath}}</string>
        <string>scan</string>
        <string>--non-interactive</string>
        <string>--out</string>
        <string>{{.OutDir}}</string>
{{- if .Profile}}
        <string>--profile</string>
        <string>{{.Profile}}</string>
{{- end}}
{{- range .ExtraArgs}}
        <string>{{.}}</string>
{{- end}}
    </array>
    <key>StartCalendarInterval</key>
    <dict>
{{- if .Weekday}}
        <key>Weekday</key>
        <integer>{{.Weekday}}</integer>
{{- end}}
{{- if .DayOfMonth}}
        <key>Day</key>
        <integer>{{.DayOfMonth}}</integer>
{{- end}}
        <key>Hour</key>
        <integer>{{.Hour}}</integer>
        <key>Minute</key>
        <integer>{{.Minute}}</integer>
    </dict>
    <key>StandardOutPath</key>
    <string>{{.StdoutLog}}</string>
    <key>StandardErrorPath</key>
    <string>{{.StderrLog}}</string>
    <key>RunAtLoad</key>
    <false/>
</dict>
</plist>
`

type launchdData struct {
	Label      string
	BinaryPath string
	OutDir     string
	Profile    string
	ExtraArgs  []string
	Weekday    int
	DayOfMonth int
	Hour       int
	Minute     int
	StdoutLog  string
	StderrLog  string
}

func (l *Launchd) plistPath(label string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist"), nil
}

// Plan implements Manager.
func (l *Launchd) Plan(opts InstallOptions) (Plan, error) {
	body, path, err := l.renderUnit(opts)
	if err != nil {
		return Plan{}, err
	}
	return Plan{
		DryRun: opts.DryRun,
		Actions: []Action{
			{Kind: "write", Path: path, Mode: 0o644, Body: body},
			{Kind: "load", Path: fmt.Sprintf("launchctl bootstrap gui/$UID %s", path)},
		},
	}, nil
}

// Install implements Manager.
func (l *Launchd) Install(ctx context.Context, opts InstallOptions) (Plan, error) {
	plan, err := l.Plan(opts)
	if err != nil {
		return Plan{}, err
	}
	if opts.DryRun {
		return plan, nil
	}
	body, path, err := l.renderUnit(opts)
	if err != nil {
		return plan, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return plan, fmt.Errorf("create LaunchAgents dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return plan, fmt.Errorf("write %s: %w", path, err)
	}
	if err := l.launchctlLoad(ctx, path); err != nil {
		return plan, err
	}
	return plan, nil
}

// Uninstall implements Manager.
func (l *Launchd) Uninstall(ctx context.Context, opts UninstallOptions) (Plan, error) {
	label := labelOrDefault(opts.Label)
	path, err := l.plistPath(label)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{
		DryRun: opts.DryRun,
		Actions: []Action{
			{Kind: "unload", Path: fmt.Sprintf("launchctl bootout gui/$UID %s", path)},
			{Kind: "remove", Path: path},
		},
	}
	if opts.DryRun {
		return plan, nil
	}
	_ = l.launchctlUnload(ctx, path) // ignore "not loaded" errors
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return plan, fmt.Errorf("remove %s: %w", path, err)
	}
	return plan, nil
}

func (l *Launchd) renderUnit(opts InstallOptions) (body, path string, err error) {
	if err := opts.Cadence.Validate(); err != nil {
		return "", "", err
	}
	weekday, dayOfMonth := scheduleTime(opts.Cadence)
	label := labelOrDefault(opts.Label)
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	outDir := opts.OutDir
	if outDir == "" {
		outDir = defaultOutDir(home)
	}

	data := launchdData{
		Label:      label,
		BinaryPath: opts.BinaryPath,
		OutDir:     outDir,
		Profile:    opts.Profile,
		ExtraArgs:  opts.ExtraArgs,
		Weekday:    weekday,
		DayOfMonth: dayOfMonth,
		Hour:       6,
		Minute:     0,
		StdoutLog:  filepath.Join(home, ".config", "osaat", "logs", "launchd-stdout.log"),
		StderrLog:  filepath.Join(home, ".config", "osaat", "logs", "launchd-stderr.log"),
	}

	tmpl, err := template.New("launchd").Parse(launchdTemplateText)
	if err != nil {
		return "", "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", "", err
	}
	path, err = l.plistPath(label)
	if err != nil {
		return "", "", err
	}
	return buf.String(), path, nil
}

func (l *Launchd) launchctlLoad(ctx context.Context, path string) error {
	if !collectors.LookupExe("launchctl") {
		return fmt.Errorf("launchctl not found on PATH; cannot load %s", path)
	}
	uid := fmt.Sprintf("%d", os.Getuid())
	_, err := collectors.DefaultRunCmd(ctx, "launchctl", "bootstrap", "gui/"+uid, path)
	return err
}

func (l *Launchd) launchctlUnload(ctx context.Context, path string) error {
	if !collectors.LookupExe("launchctl") {
		return nil
	}
	uid := fmt.Sprintf("%d", os.Getuid())
	_, err := collectors.DefaultRunCmd(ctx, "launchctl", "bootout", "gui/"+uid, path)
	return err
}

// labelOrDefault expands a missing label to the canonical
// "com.simtabi.osaat" reverse-DNS form used by other Simtabi tools.
func labelOrDefault(label string) string {
	if label != "" {
		return label
	}
	return "com.simtabi.osaat"
}
