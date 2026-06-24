// Package tui implements the interactive terminal UI for mta. The home screen is
// a navigable menu (↑/↓ or j/k, Enter to select, Esc to go back); from it you
// reach the live status, manual overrides, the schedule and settings editors,
// service control, the Graph account, updates, and help.
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

// Run starts the interactive UI.
func Run(opts Options) error {
	p := tea.NewProgram(newModel(opts), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

type tickMsg time.Time
type updateMsg struct{ info selfupdate.Info }

type screen int

const (
	screenMenu screen = iota
	screenDashboard
	screenOverride
	screenSchedule
	screenSettings
	screenService
	screenAccount
	screenHelp
	screenOnboard
)

// menuID identifies a main-menu entry.
type menuID int

const (
	miStatus menuID = iota
	miOverride
	miSchedule
	miSettings
	miService
	miAccount
	miUpdate
	miHelp
	miQuit
)

type menuEntry struct {
	id    menuID
	title string
	desc  string
}

func mainMenu() []menuEntry {
	return []menuEntry{
		{miStatus, "Status", "Live daemon status and recent activity"},
		{miOverride, "Override", "Force active/inactive now, or resume the schedule"},
		{miSchedule, "Schedule", "Edit the weekly active windows"},
		{miSettings, "Settings", "Engine, interval, movement, timezone, Graph"},
		{miService, "Service", "Install / start / stop / restart the background service"},
		{miAccount, "Account", "Microsoft Graph sign-in (for the graph engine)"},
		{miUpdate, "Check for updates", "Download and install the latest release"},
		{miHelp, "Help", "Keys and what everything does"},
		{miQuit, "Quit", "Exit the dashboard"},
	}
}

type model struct {
	opts   Options
	exe    string
	status control.Status
	stErr  error
	cfg    config.Config
	cfgErr error
	logs   []string
	flash  string
	update selfupdate.Info

	screen screen
	menu   []menuEntry
	cursor int // main-menu cursor

	subCursor int // override/service/account submenu cursor

	// Schedule editor state.
	edit   config.Config
	winIdx int
	field  int

	// Settings editor state.
	setRow     int
	setInput   textinput.Model
	setEditing bool
}

func newModel(opts Options) model {
	exe, _ := os.Executable()
	ti := textinput.New()
	ti.CharLimit = 128
	m := model{opts: opts, exe: exe, setInput: ti, menu: mainMenu(), screen: screenMenu}
	m.refresh()
	if m.cfgErr != nil {
		m.screen = screenOnboard
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

// execSelf suspends the TUI, runs `mta <args>`, then resumes and refreshes.
func (m model) execSelf(args ...string) tea.Cmd {
	c := exec.Command(m.exe, args...)
	return tea.ExecProcess(c, func(error) tea.Msg { return tickMsg(time.Now()) })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil // views are width-agnostic
	case updateMsg:
		m.update = msg.info
		return m, nil
	case tickMsg:
		m.refresh()
		return m, tick()
	case tea.KeyMsg:
		switch m.screen {
		case screenMenu:
			return m.updateMenu(msg)
		case screenOverride:
			return m.updateOverride(msg)
		case screenSchedule:
			return m.updateEditor(msg)
		case screenSettings:
			return m.updateSettings(msg)
		case screenService:
			return m.updateService(msg)
		case screenAccount:
			return m.updateAccount(msg)
		case screenOnboard:
			return m.updateOnboard(msg)
		case screenDashboard, screenHelp:
			if back(msg) {
				m.screen = screenMenu
			}
			return m, nil
		}
	}
	return m, nil
}

// back reports whether a key means "go back / dismiss".
func back(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "esc", "q", "enter", "backspace", "left", "h":
		return true
	}
	return false
}

// moveCursor clamps a cursor within [0, n).
func moveCursor(cur, n, delta int) int {
	cur += delta
	if cur < 0 {
		cur = 0
	}
	if cur > n-1 {
		cur = n - 1
	}
	return cur
}

func (m model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		m.cursor = moveCursor(m.cursor, len(m.menu), -1)
	case "down", "j":
		m.cursor = moveCursor(m.cursor, len(m.menu), +1)
	case "enter", "right", "l":
		return m.selectMenu(m.menu[m.cursor].id)
	}
	return m, nil
}

func (m model) selectMenu(id menuID) (tea.Model, tea.Cmd) {
	m.flash = ""
	m.subCursor = 0
	switch id {
	case miStatus:
		m.screen = screenDashboard
	case miOverride:
		m.screen = screenOverride
	case miSchedule:
		m.enterEditor()
	case miSettings:
		m.enterSettings()
	case miService:
		m.screen = screenService
	case miAccount:
		m.screen = screenAccount
	case miUpdate:
		if selfupdate.IsDev(m.opts.Version) {
			m.flash = "this is a dev build; install a released version to self-update"
			return m, nil
		}
		return m, m.execSelf("upgrade")
	case miHelp:
		m.screen = screenHelp
	case miQuit:
		return m, tea.Quit
	}
	return m, nil
}

// ---- Override submenu ----

var overrideItems = []struct{ key, label, desc string }{
	{"on", "Force active", "Stay Available now, ignoring the schedule"},
	{"off", "Force inactive", "Stop keeping active now"},
	{"resume", "Resume schedule", "Clear the override and follow the schedule"},
	{"back", "Back", ""},
}

func (m model) updateOverride(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenMenu
	case "up", "k":
		m.subCursor = moveCursor(m.subCursor, len(overrideItems), -1)
	case "down", "j":
		m.subCursor = moveCursor(m.subCursor, len(overrideItems), +1)
	case "enter", "right", "l":
		switch overrideItems[m.subCursor].key {
		case "on":
			m.setOverride(schedule.OverrideOn)
		case "off":
			m.setOverride(schedule.OverrideOff)
		case "resume":
			if err := os.Remove(control.OverridePath(m.opts.RuntimeDir)); err != nil && !os.IsNotExist(err) {
				m.flash = "resume failed: " + err.Error()
			} else {
				m.flash = "override cleared; following schedule"
			}
			m.refresh()
		}
		m.screen = screenMenu
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

// ---- Account submenu ----

var accountItems = []struct {
	args        []string
	label, desc string
}{
	{[]string{"auth", "login"}, "Sign in", "Microsoft device-code sign-in"},
	{[]string{"auth", "status"}, "Status", "Show the signed-in account"},
	{[]string{"auth", "logout"}, "Sign out", "Remove cached credentials"},
	{nil, "Back", ""},
}

func (m model) updateAccount(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenMenu
	case "up", "k":
		m.subCursor = moveCursor(m.subCursor, len(accountItems), -1)
	case "down", "j":
		m.subCursor = moveCursor(m.subCursor, len(accountItems), +1)
	case "enter", "right", "l":
		it := accountItems[m.subCursor]
		m.screen = screenMenu
		if it.args == nil {
			return m, nil
		}
		return m, m.execSelf(it.args...)
	}
	return m, nil
}

// ---- Onboarding ----

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
		m.screen = screenMenu
		m.flash = "wrote default config"
	case "w":
		return m, m.execSelf("config", "wizard")
	}
	return m, nil
}

// ---- Styles ----

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	boxStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Margin(0, 0, 1, 0)
	labelStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	activeStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	idleStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	flashStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	bannerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("42")).Padding(0, 1)
	selRowStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)
	selFieldStyle = lipgloss.NewStyle().Background(lipgloss.Color("63")).Foreground(lipgloss.Color("231"))
	dayOnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	dayOffStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func (m model) View() string {
	switch m.screen {
	case screenOverride:
		return m.menuFrame("Override", listView(overrideItemsLabels(), m.subCursor))
	case screenAccount:
		return m.menuFrame("Account", listView(accountItemsLabels(), m.subCursor))
	case screenService:
		return m.menuFrame("Service", m.serviceBody())
	case screenSchedule:
		return m.editorView()
	case screenSettings:
		return m.settingsView()
	case screenHelp:
		return m.helpView()
	case screenOnboard:
		return m.onboardView()
	case screenDashboard:
		return m.menuFrame("Status", m.statusView()+"\n\n"+m.logView())
	default:
		return m.menuView()
	}
}

// header renders the title line + optional update banner + status strip.
func (m model) header() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("  MS Teams Activity") + "  " +
		helpStyle.Render("scope="+string(m.opts.Scope)+"  v"+m.opts.Version) + "\n")
	if m.update.Available {
		b.WriteString("\n" + bannerStyle.Render(fmt.Sprintf("update available: %s → %s",
			m.update.Current, m.update.Latest)) + "\n")
	}
	b.WriteString(helpStyle.Render("  "+m.statusStrip()) + "\n\n")
	return b.String()
}

func (m model) statusStrip() string {
	if m.stErr != nil {
		return "daemon: not running"
	}
	state := "idle"
	if m.status.DesiredActive {
		state = "ACTIVE"
	}
	s := "state: " + state + "   engine: " + m.status.Engine
	if m.status.OverrideMode != "" {
		s += "   override: " + m.status.OverrideMode
	}
	if m.status.NextChange != nil {
		word := "deactivate"
		if m.status.NextActive {
			word = "activate"
		}
		s += "   next: " + word + " " + m.status.NextChange.Format("Mon 15:04")
	}
	return s
}

func (m model) menuView() string {
	var b strings.Builder
	b.WriteString(m.header())
	for i, e := range m.menu {
		cursor := "   "
		line := e.title
		if i == m.cursor {
			cursor = selRowStyle.Render(" ▸ ")
			line = selRowStyle.Render(e.title) + "  " + helpStyle.Render(e.desc)
		} else {
			line = line + "  " + helpStyle.Render(e.desc)
		}
		b.WriteString(cursor + line + "\n")
	}
	if m.flash != "" {
		b.WriteString("\n" + flashStyle.Render("• "+m.flash) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("↑/↓ move · enter select · q quit"))
	return b.String()
}

// menuFrame wraps a submenu/detail screen with the shared header + footer.
func (m model) menuFrame(title, body string) string {
	var b strings.Builder
	b.WriteString(m.header())
	b.WriteString(titleStyle.Render("  "+title) + "\n\n")
	b.WriteString(body + "\n")
	if m.flash != "" {
		b.WriteString("\n" + flashStyle.Render("• "+m.flash) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("↑/↓ move · enter select · esc back"))
	return b.String()
}

// listView renders a simple cursor list from labels.
func listView(labels []string, cursor int) string {
	var lines []string
	for i, l := range labels {
		if i == cursor {
			lines = append(lines, selRowStyle.Render(" ▸ "+l))
		} else {
			lines = append(lines, "   "+l)
		}
	}
	return strings.Join(lines, "\n")
}

func overrideItemsLabels() []string {
	out := make([]string, len(overrideItems))
	for i, it := range overrideItems {
		out[i] = it.label
		if it.desc != "" {
			out[i] += "  " + helpStyle.Render(it.desc)
		}
	}
	return out
}

func accountItemsLabels() []string {
	out := make([]string, len(accountItems))
	for i, it := range accountItems {
		out[i] = it.label
		if it.desc != "" {
			out[i] += "  " + helpStyle.Render(it.desc)
		}
	}
	return out
}

func (m model) helpView() string {
	body := strings.Join([]string{
		"Navigate with ↑/↓ (or j/k); Enter selects; Esc goes back; q quits.",
		"",
		"  Status    live daemon state + recent log",
		"  Override  force active/inactive, or resume the schedule",
		"  Schedule  edit weekly windows (days, start, end)",
		"  Settings  engine, interval, movement, timezone, Graph app",
		"  Service   install/start/stop/restart the background service",
		"  Account   Microsoft Graph sign-in for the graph engine",
		"",
		"Everything here is also available as CLI commands — see `mta --help`.",
	}, "\n")
	return m.menuFrame("Help", body)
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
		out[i] = "  " + l
	}
	return out
}
