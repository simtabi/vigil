// Package tui implements the interactive terminal dashboard for mta: live
// status, manual override controls and a tail of the daemon log.
package tui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/simtabi/ms-teams-activity/internal/config"
	"github.com/simtabi/ms-teams-activity/internal/control"
	"github.com/simtabi/ms-teams-activity/internal/schedule"
)

// Options configures the TUI with the resolved paths for the active scope.
type Options struct {
	Scope      config.Scope
	ConfigPath string
	RuntimeDir string
}

// Run starts the Bubble Tea program and blocks until the user quits.
func Run(opts Options) error {
	p := tea.NewProgram(newModel(opts), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

type tickMsg time.Time

type model struct {
	opts   Options
	status control.Status
	stErr  error
	cfg    config.Config
	cfgErr error
	logs   []string
	flash  string
	width  int
	height int
}

func newModel(opts Options) model {
	m := model{opts: opts}
	m.refresh()
	return m
}

func (m model) Init() tea.Cmd { return tick() }

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *model) refresh() {
	m.status, m.stErr = control.ReadStatus(m.opts.RuntimeDir)
	m.cfg, m.cfgErr = config.Load(m.opts.ConfigPath)
	m.logs = tailLines(control.LogPath(m.opts.RuntimeDir), 8)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		m.refresh()
		return m, tick()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "o":
			m.setOverride(schedule.OverrideOn)
		case "f":
			m.setOverride(schedule.OverrideOff)
		case "r":
			if err := os.Remove(control.OverridePath(m.opts.RuntimeDir)); err != nil && !os.IsNotExist(err) {
				m.flash = "resume failed: " + err.Error()
			} else {
				m.flash = "override cleared"
			}
			m.refresh()
		}
	}
	return m, nil
}

func (m *model) setOverride(mode schedule.OverrideMode) {
	ov := schedule.Override{Mode: mode, SetAt: time.Now()}
	if err := schedule.SaveOverride(control.OverridePath(m.opts.RuntimeDir), ov); err != nil {
		m.flash = "override failed: " + err.Error()
		return
	}
	m.flash = "override set: " + string(mode)
	m.refresh()
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	boxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Margin(0, 0, 1, 0)
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	activeStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	idleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	flashStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
)

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("  MS Teams Activity") + "  " + helpStyle.Render("scope="+string(m.opts.Scope)) + "\n\n")

	b.WriteString(boxStyle.Render(m.statusView()) + "\n")
	b.WriteString(boxStyle.Render(m.scheduleView()) + "\n")
	b.WriteString(boxStyle.Render(m.logView()) + "\n")

	if m.flash != "" {
		b.WriteString(flashStyle.Render("• "+m.flash) + "\n")
	}
	b.WriteString(helpStyle.Render("[o] force on   [f] force off   [r] resume schedule   [q] quit"))
	return b.String()
}

func (m model) statusView() string {
	if m.stErr != nil {
		return labelStyle.Render("Daemon: ") + m.stErr.Error()
	}
	state := idleStyle.Render("idle")
	if m.status.DesiredActive {
		state = activeStyle.Render("ACTIVE")
	}
	lines := []string{
		labelStyle.Render("State:    ") + state,
		labelStyle.Render("Engine:   ") + m.status.Engine,
	}
	if len(m.status.Activators) > 0 {
		lines = append(lines, labelStyle.Render("Drivers:  ")+strings.Join(m.status.Activators, ", "))
	}
	if m.status.OverrideMode != "" {
		ov := "override " + m.status.OverrideMode
		if m.status.OverrideUntil != nil {
			ov += " until " + m.status.OverrideUntil.Format("Mon 15:04")
		}
		lines = append(lines, labelStyle.Render("Override: ")+ov)
	}
	if m.status.NextChange != nil {
		word := "deactivate"
		if m.status.NextActive {
			word = "activate"
		}
		lines = append(lines, labelStyle.Render("Next:     ")+word+" at "+m.status.NextChange.Format("Mon 15:04"))
	}
	if m.status.LastError != "" {
		lines = append(lines, labelStyle.Render("Error:    ")+m.status.LastError)
	}
	return strings.Join(lines, "\n")
}

func (m model) scheduleView() string {
	if m.cfgErr != nil {
		return labelStyle.Render("Schedule: ") + m.cfgErr.Error()
	}
	if m.cfg.Schedule.Always {
		return labelStyle.Render("Schedule: ") + "always on"
	}
	if !m.cfg.Schedule.Enabled {
		return labelStyle.Render("Schedule: ") + "disabled (override only)"
	}
	lines := []string{labelStyle.Render("Schedule (tz " + m.cfg.Timezone + "):")}
	for _, w := range m.cfg.Schedule.Windows {
		lines = append(lines, "  "+strings.Join(w.Days, " ")+"  "+w.Start+"–"+w.End)
	}
	lines = append(lines, helpStyle.Render("  edit with: mta config edit"))
	return strings.Join(lines, "\n")
}

func (m model) logView() string {
	if len(m.logs) == 0 {
		return labelStyle.Render("Log: ") + "(no entries yet)"
	}
	return labelStyle.Render("Recent log:") + "\n" + strings.Join(m.logs, "\n")
}

// tailLines returns up to n trailing lines of the file at path.
func tailLines(path string, n int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var all []string
	for sc.Scan() {
		all = append(all, sc.Text())
	}
	if len(all) > n {
		all = all[len(all)-n:]
	}
	out := make([]string, len(all))
	for i, l := range all {
		if len(l) > 110 {
			l = l[:107] + "..."
		}
		out[i] = fmt.Sprintf("  %s", l)
	}
	return out
}
