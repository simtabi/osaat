package schedule

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/simtabi/osaat/internal/collectors"
)

// Systemd implements Manager for Linux via systemd user units. The
// generated pair is osaat.service (oneshot) + osaat.timer (OnCalendar).
type Systemd struct{}

// NewSystemd returns a systemd Manager.
func NewSystemd() *Systemd { return &Systemd{} }

const serviceTemplateText = `[Unit]
Description=osaat application audit
Documentation=https://opensource.simtabi.com/documentation/osaat

[Service]
Type=oneshot
ExecStart={{.BinaryPath}} scan --non-interactive --out {{.OutDir}}{{range .ExtraArgs}} {{.}}{{end}}{{if .Profile}} --profile {{.Profile}}{{end}}
Environment=HOME=%h
StandardOutput=append:%h/.config/osaat/logs/systemd-stdout.log
StandardError=append:%h/.config/osaat/logs/systemd-stderr.log
`

const timerTemplateText = `[Unit]
Description=Run osaat scan on a {{.Cadence}} cadence
Documentation=https://opensource.simtabi.com/documentation/osaat

[Timer]
OnCalendar={{.OnCalendar}}
Persistent=true
Unit={{.ServiceName}}

[Install]
WantedBy=timers.target
`

type systemdData struct {
	BinaryPath  string
	OutDir      string
	Profile     string
	ExtraArgs   []string
	Cadence     string
	OnCalendar  string
	ServiceName string
}

func (s *Systemd) unitDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "systemd", "user"), nil
}

func (s *Systemd) unitPaths(label string) (servicePath, timerPath string, err error) {
	dir, err := s.unitDir()
	if err != nil {
		return "", "", err
	}
	base := labelOrSystemd(label)
	return filepath.Join(dir, base+".service"), filepath.Join(dir, base+".timer"), nil
}

// Plan implements Manager.
func (s *Systemd) Plan(opts InstallOptions) (Plan, error) {
	srvBody, timerBody, srvPath, timerPath, err := s.renderUnits(opts)
	if err != nil {
		return Plan{}, err
	}
	base := labelOrSystemd(opts.Label)
	return Plan{
		DryRun: opts.DryRun,
		Actions: []Action{
			{Kind: "write", Path: srvPath, Mode: 0o644, Body: srvBody},
			{Kind: "write", Path: timerPath, Mode: 0o644, Body: timerBody},
			{Kind: "load", Path: fmt.Sprintf("systemctl --user daemon-reload && systemctl --user enable --now %s.timer", base)},
		},
	}, nil
}

// Install implements Manager.
func (s *Systemd) Install(ctx context.Context, opts InstallOptions) (Plan, error) {
	plan, err := s.Plan(opts)
	if err != nil {
		return Plan{}, err
	}
	if opts.DryRun {
		return plan, nil
	}
	srvBody, timerBody, srvPath, timerPath, err := s.renderUnits(opts)
	if err != nil {
		return plan, err
	}
	if err := os.MkdirAll(filepath.Dir(srvPath), 0o755); err != nil {
		return plan, fmt.Errorf("create systemd user dir: %w", err)
	}
	if err := os.WriteFile(srvPath, []byte(srvBody), 0o644); err != nil {
		return plan, fmt.Errorf("write %s: %w", srvPath, err)
	}
	if err := os.WriteFile(timerPath, []byte(timerBody), 0o644); err != nil {
		return plan, fmt.Errorf("write %s: %w", timerPath, err)
	}
	if err := s.systemctlLoad(ctx, opts.Label); err != nil {
		return plan, err
	}
	return plan, nil
}

// Uninstall implements Manager.
func (s *Systemd) Uninstall(ctx context.Context, opts UninstallOptions) (Plan, error) {
	srvPath, timerPath, err := s.unitPaths(opts.Label)
	if err != nil {
		return Plan{}, err
	}
	base := labelOrSystemd(opts.Label)
	plan := Plan{
		DryRun: opts.DryRun,
		Actions: []Action{
			{Kind: "unload", Path: fmt.Sprintf("systemctl --user disable --now %s.timer", base)},
			{Kind: "remove", Path: timerPath},
			{Kind: "remove", Path: srvPath},
		},
	}
	if opts.DryRun {
		return plan, nil
	}
	_ = s.systemctlUnload(ctx, base)
	for _, p := range []string{timerPath, srvPath} {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return plan, fmt.Errorf("remove %s: %w", p, err)
		}
	}
	return plan, nil
}

func (s *Systemd) renderUnits(opts InstallOptions) (srvBody, timerBody, srvPath, timerPath string, err error) {
	if err = opts.Cadence.Validate(); err != nil {
		return
	}
	home, herr := os.UserHomeDir()
	if herr != nil {
		err = herr
		return
	}
	outDir := opts.OutDir
	if outDir == "" {
		outDir = defaultOutDir(home)
	}

	base := labelOrSystemd(opts.Label)
	data := systemdData{
		BinaryPath:  opts.BinaryPath,
		OutDir:      outDir,
		Profile:     opts.Profile,
		ExtraArgs:   opts.ExtraArgs,
		Cadence:     string(opts.Cadence),
		OnCalendar:  onCalendarFor(opts.Cadence),
		ServiceName: base + ".service",
	}

	srvTmpl, perr := template.New("svc").Parse(serviceTemplateText)
	if perr != nil {
		err = perr
		return
	}
	var srvBuf bytes.Buffer
	if perr := srvTmpl.Execute(&srvBuf, data); perr != nil {
		err = perr
		return
	}
	srvBody = srvBuf.String()

	timerTmpl, perr := template.New("timer").Parse(timerTemplateText)
	if perr != nil {
		err = perr
		return
	}
	var timerBuf bytes.Buffer
	if perr := timerTmpl.Execute(&timerBuf, data); perr != nil {
		err = perr
		return
	}
	timerBody = timerBuf.String()

	srvPath, timerPath, err = s.unitPaths(opts.Label)
	return
}

func (s *Systemd) systemctlLoad(ctx context.Context, label string) error {
	if !collectors.LookupExe("systemctl") {
		return fmt.Errorf("systemctl not found on PATH; cannot enable timer")
	}
	if _, err := collectors.DefaultRunCmd(ctx, "systemctl", "--user", "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}
	base := labelOrSystemd(label)
	_, err := collectors.DefaultRunCmd(ctx, "systemctl", "--user", "enable", "--now", base+".timer")
	return err
}

func (s *Systemd) systemctlUnload(ctx context.Context, base string) error {
	if !collectors.LookupExe("systemctl") {
		return nil
	}
	_, _ = collectors.DefaultRunCmd(ctx, "systemctl", "--user", "disable", "--now", base+".timer")
	return nil
}

// labelOrSystemd returns the systemd unit basename (without extension)
// — defaults to "osaat".
func labelOrSystemd(label string) string {
	label = strings.TrimSuffix(label, ".timer")
	label = strings.TrimSuffix(label, ".service")
	if label == "" {
		return "osaat"
	}
	return label
}

// onCalendarFor maps a Cadence to systemd's OnCalendar syntax. Time
// is always 06:00 local — matches the launchd defaults.
func onCalendarFor(c Cadence) string {
	switch c {
	case CadenceDaily:
		return "*-*-* 06:00:00"
	case CadenceWeekly:
		return "Mon *-*-* 06:00:00"
	case CadenceMonthly:
		return "*-*-01 06:00:00"
	default:
		return "weekly"
	}
}
