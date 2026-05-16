package wizard

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PhaseMsg moves the live scan view to a new phase label. Send it
// from the scanning code via Program.Send.
type PhaseMsg struct {
	Phase string
}

// CountMsg adds or replaces a counter line in the live scan view.
type CountMsg struct {
	Key   string
	Count int
}

// DoneMsg ends the live view and prints the closing line. Sending it
// also triggers tea.Quit.
type DoneMsg struct {
	Duration time.Duration
	Err      error
}

// tickMsg is the internal one-hertz tick that drives the elapsed-time
// readout.
type tickMsg time.Time

// ScanModel is the bubbletea model that drives the live scan view.
type ScanModel struct {
	phase    string
	counts   []counter
	started  time.Time
	elapsed  time.Duration
	done     bool
	err      error
	width    int
}

type counter struct {
	key   string
	count int
}

// NewScanModel returns a fresh scan model. Start it with
// tea.NewProgram(model).Run() in a goroutine, then send PhaseMsg /
// CountMsg / DoneMsg from the scanning code.
func NewScanModel() *ScanModel {
	return &ScanModel{
		phase:   "Starting…",
		started: time.Now(),
	}
}

// Init implements tea.Model.
func (m *ScanModel) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Update implements tea.Model.
func (m *ScanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case PhaseMsg:
		m.phase = msg.Phase
		return m, nil
	case CountMsg:
		for i := range m.counts {
			if m.counts[i].key == msg.Key {
				m.counts[i].count = msg.Count
				return m, nil
			}
		}
		m.counts = append(m.counts, counter{key: msg.Key, count: msg.Count})
		return m, nil
	case DoneMsg:
		m.done = true
		m.err = msg.Err
		m.elapsed = msg.Duration
		return m, tea.Quit
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case tickMsg:
		if !m.done {
			m.elapsed = time.Since(m.started).Round(time.Second)
			return m, tickCmd()
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m *ScanModel) View() string {
	header := lipgloss.NewStyle().Bold(true).Render("osaat — live scan")
	phaseLine := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render("► " + m.phase)
	elapsed := fmt.Sprintf("elapsed: %s", m.elapsed)

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n\n")
	sb.WriteString(phaseLine)
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render(elapsed))
	sb.WriteString("\n")

	if len(m.counts) > 0 {
		sb.WriteString("\n")
		for _, c := range m.counts {
			sb.WriteString(fmt.Sprintf("  %-20s %d\n", c.key, c.count))
		}
	}

	if m.done {
		sb.WriteString("\n")
		if m.err != nil {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗ scan failed: " + m.err.Error()))
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Render("✓ scan complete"))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
