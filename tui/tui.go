// Package tui implements the interactive terminal hub for mta: live status,
// manual overrides, schedule editor, settings editor, service/auth/update
// actions, and first-run onboarding.
package tui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/simtabi/ms-teams-activity/internal/config"
	"github.com/simtabi/ms-teams-activity/internal/control"
	"github.com/simtabi/ms-teams-activity/internal/schedule"
	"github.com/simtabi/ms-teams-activity/internal/selfupdate"
)

// Options configures the TUI with resolved paths for the active scope.
type Options struct {
	Scope      config.Scope
	ConfigPath string
	RuntimeDir string
	Version    string
}

// Run starts the dashboard hub.
func Run(opts Options) error {
	p := tea.NewProgram(newModel(opts), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

type tickMsg time.Time
type updateMsg struct{ info selfupdate.Info }

type viewMode int

const (
	modeDashboard viewMode = iota
	modeEditor
	modeSettings
	modeService
	modeHelp
	modeOnboard
)

type model struct {
	opts   Options
	exe    string
	status control.Status
	stErr  error
	cfg    config.Config
	cfgErr error
	logs   []string
	flash  string
	mode   viewMode
	update selfupdate.Info

	// Schedule editor state.
	edit   config.Config
	winIdx int
	field  int

	// Settings editor state.
	setRow     int
	setInput   textinput.Model
	setEditing bool

	// Service menu state.
	svcRow int
}

func newModel(opts Options) model {
	exe, _ := os.Executable()
	ti := textinput.New()
	ti.CharLimit = 128
	m := model{opts: opts, exe: exe, setInput: ti}
	m.refresh()
	if m.cfgErr != nil {
		m.mode = modeOnboard
	}
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tick(), checkUpdateCmd(m.opts.Version))
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func checkUpdateCmd(version string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		info, err := selfupdate.Check(ctx, version)
		if err != nil {
			return updateMsg{}
		}
		return updateMsg{info: info}
	}
}

func (m *model) refresh() {
	m.status, m.stErr = control.ReadStatus(m.opts.RuntimeDir)
	m.cfg, m.cfgErr = config.Load(m.opts.ConfigPath)
	m.logs = tailLines(control.LogPath(m.opts.RuntimeDir), 6)
}

// exec suspends the TUI, runs `mta <args>`, then resumes and refreshes.
func (m model) execSelf(args ...string) tea.Cmd {
	c := exec.Command(m.exe, args...)
	return tea.ExecProcess(c, func(error) tea.Msg { return tickMsg(time.Now()) })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// no layout state needed; views are width-agnostic
	case updateMsg:
		m.update = msg.info
		return m, nil
	case tickMsg:
		m.refresh()
		return m, tick()
	case tea.KeyMsg:
		switch m.mode {
		case modeEditor:
			return m.updateEditor(msg)
		case modeSettings:
			return m.updateSettings(msg)
		case modeService:
			return m.updateService(msg)
		case modeOnboard:
			return m.updateOnboard(msg)
		case modeHelp:
			m.mode = modeDashboard
			return m, nil
		default:
			return m.updateDashboard(msg)
		}
	}
	return m, nil
}

func (m model) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
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
	case "e":
		m.enterEditor()
	case "c":
		m.enterSettings()
	case "v":
		m.mode = modeService
		m.svcRow = 0
	case "a":
		return m, m.execSelf("auth", "login")
	case "u":
		if m.update.Available {
			return m, m.execSelf("upgrade", "--yes")
		}
		m.flash = "no update available"
	case "?":
		m.mode = modeHelp
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

func (m model) updateOnboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "i":
		if err := config.Default().Save(m.opts.ConfigPath); err != nil {
			m.flash = "init failed: " + err.Error()
			return m, nil
		}
		m.refresh()
		m.mode = modeDashboard
		m.flash = "wrote default config"
	case "w":
		return m, m.execSelf("config", "wizard")
	}
	return m, nil
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	boxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Margin(0, 0, 1, 0)
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	activeStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	idleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	flashStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	bannerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("42")).Padding(0, 1)
)

func (m model) View() string {
	switch m.mode {
	case modeEditor:
		return m.editorView()
	case modeSettings:
		return m.settingsView()
	case modeService:
		return m.serviceView()
	case modeHelp:
		return m.helpView()
	case modeOnboard:
		return m.onboardView()
	default:
		return m.dashboardView()
	}
}

func (m model) dashboardView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("  MS Teams Activity") + "  " + helpStyle.Render("scope="+string(m.opts.Scope)+"  v"+m.opts.Version) + "\n\n")
	if m.update.Available {
		b.WriteString(bannerStyle.Render(fmt.Sprintf("update available: %s → %s  (press u)", m.update.Current, m.update.Latest)) + "\n\n")
	}
	b.WriteString(boxStyle.Render(m.statusView()) + "\n")
	b.WriteString(boxStyle.Render(m.scheduleView()) + "\n")
	b.WriteString(boxStyle.Render(m.logView()) + "\n")
	if m.flash != "" {
		b.WriteString(flashStyle.Render("• "+m.flash) + "\n")
	}
	b.WriteString(helpStyle.Render("[o]n [f]off [r]esume  [e]schedule [c]onfig [v]service [a]uth [u]pdate  [?]help [q]uit"))
	return b.String()
}

func (m model) onboardView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("  MS Teams Activity — first run") + "\n\n")
	b.WriteString(boxStyle.Render("No config found at:\n  "+m.opts.ConfigPath+"\n\nLet's set it up.") + "\n")
	if m.flash != "" {
		b.WriteString(flashStyle.Render("• "+m.flash) + "\n")
	}
	b.WriteString(helpStyle.Render("[w] guided setup wizard   [i] write defaults   [q] quit"))
	return b.String()
}

func (m model) helpView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("  Help") + "\n\n")
	body := strings.Join([]string{
		"Dashboard:",
		"  o / f / r   override on / off / resume schedule",
		"  e           edit schedule windows",
		"  c           edit settings (engine, interval, graph, ...)",
		"  v           service actions (install/start/stop/...)",
		"  a           sign in to Microsoft Graph",
		"  u           install available update",
		"  q           quit",
		"",
		"Everything here is also available as CLI commands (see `mta --help`).",
	}, "\n")
	b.WriteString(boxStyle.Render(body) + "\n")
	b.WriteString(helpStyle.Render("press any key to return"))
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
