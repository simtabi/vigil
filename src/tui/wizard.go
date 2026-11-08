package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/simtabi/vigil/internal/config"
)

// RunWizard launches the guided setup wizard and writes config on completion.
func RunWizard(opts Options) error {
	wm := newWizard(opts)
	final, err := tea.NewProgram(wm, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	if w, ok := final.(wizardModel); ok {
		if w.err != nil {
			return w.err
		}
		if w.saved {
			fmt.Printf("config written to %s\n", opts.ConfigPath)
		}
	}
	return nil
}

const (
	wEngine = iota
	wTimezone
	wPreset
	wClientID
	wTenantID
	wConfirm
)

type wizardModel struct {
	opts   Options
	step   int
	cfg    config.Config
	cursor int
	input  textinput.Model
	tz     picker
	saved  bool
	err    error
	flash  string
}

var engineChoices = []config.Engine{config.EngineInput, config.EngineGraph, config.EngineBoth}

type presetChoice struct {
	label string
	apply func(c *config.Config)
}

var presetChoices = []presetChoice{
	{"Mon–Fri 08:00–17:00", func(c *config.Config) {
		c.Schedule = config.ScheduleConfig{Enabled: true, Windows: []config.Window{
			{Days: []string{"Mon", "Tue", "Wed", "Thu", "Fri"}, Start: "08:00", End: "17:00"},
		}}
	}},
	{"Always on (24/7)", func(c *config.Config) {
		c.Schedule = config.ScheduleConfig{Enabled: true, Always: true}
	}},
	{"Manual only (overrides)", func(c *config.Config) {
		c.Schedule = config.ScheduleConfig{Enabled: false}
	}},
}

func newWizard(opts Options) wizardModel {
	ti := textinput.New()
	ti.CharLimit = 128
	cfg := config.Default()
	if existing, err := config.Load(opts.ConfigPath); err == nil {
		cfg = existing
	}
	return wizardModel{opts: opts, cfg: cfg, input: ti, tz: newPicker("Timezone (or 'Local')", cfg.Timezone)}
}

func (m wizardModel) Init() tea.Cmd { return nil }

func (m wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if key.String() == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.step {
	case wEngine:
		return m.choose(key, len(engineChoices), func(i int) {
			m.cfg.Engine = engineChoices[i]
		})
	case wPreset:
		return m.choose(key, len(presetChoices), func(i int) {
			presetChoices[i].apply(&m.cfg)
		})
	case wTimezone:
		chosen, cancelled := m.tz.update(key)
		switch {
		case cancelled:
			m.step = m.prev()
		case chosen != "":
			m.cfg.Timezone = chosen
			m.cursor = 0
			m.step = m.next()
		}
		return m, nil
	case wClientID:
		return m.text(key, &m.cfg.Graph.ClientID, "")
	case wTenantID:
		return m.text(key, &m.cfg.Graph.TenantID, "common")
	case wConfirm:
		switch key.String() {
		case "enter", "y":
			if err := m.cfg.Validate(); err != nil {
				m.flash = "invalid: " + err.Error()
				return m, nil
			}
			if err := m.cfg.Save(m.opts.ConfigPath); err != nil {
				m.err = err
				return m, tea.Quit
			}
			m.saved = true
			return m, tea.Quit
		case "esc":
			m.step = m.prev()
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

// choose handles a cursor list step.
func (m wizardModel) choose(key tea.KeyMsg, n int, set func(i int)) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < n-1 {
			m.cursor++
		}
	case "enter":
		set(m.cursor)
		m.cursor = 0
		m.step = m.next()
	case "esc":
		m.step = m.prev()
	}
	return m, nil
}

// text handles a single text-input step writing into dst (def used as placeholder).
func (m wizardModel) text(key tea.KeyMsg, dst *string, def string) (tea.Model, tea.Cmd) {
	if !m.input.Focused() {
		m.input.SetValue(*dst)
		m.input.Placeholder = def
		m.input.Focus()
	}
	switch key.String() {
	case "enter":
		v := strings.TrimSpace(m.input.Value())
		if v == "" {
			v = def
		}
		*dst = v
		m.input.Blur()
		m.input.SetValue("")
		m.step = m.next()
		return m, nil
	case "esc":
		m.input.Blur()
		m.input.SetValue("")
		m.step = m.prev()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(key)
	return m, cmd
}

func (m wizardModel) next() int {
	switch m.step {
	case wEngine:
		return wTimezone
	case wTimezone:
		return wPreset
	case wPreset:
		if m.cfg.UsesGraph() {
			return wClientID
		}
		return wConfirm
	case wClientID:
		return wTenantID
	case wTenantID:
		return wConfirm
	}
	return wConfirm
}

func (m wizardModel) prev() int {
	switch m.step {
	case wTimezone:
		return wEngine
	case wPreset:
		return wTimezone
	case wClientID:
		return wPreset
	case wTenantID:
		return wClientID
	case wConfirm:
		if m.cfg.UsesGraph() {
			return wTenantID
		}
		return wPreset
	}
	return wEngine
}

func (m wizardModel) View() string {
	// The timezone step is a full searchable picker with its own footer.
	if m.step == wTimezone {
		return titleStyle.Render("  Setup wizard") + "\n\n" + m.tz.view()
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("  Setup wizard") + "\n\n")
	switch m.step {
	case wEngine:
		b.WriteString(renderChoices("How should Teams be kept active?", engineLabels(), m.cursor))
	case wPreset:
		b.WriteString(renderChoices("Pick a schedule:", presetLabels(), m.cursor))
	case wClientID:
		b.WriteString("Microsoft Entra application (client) ID:\n\n" + m.input.View())
	case wTenantID:
		b.WriteString("Tenant (GUID, 'common', or 'organizations'):\n\n" + m.input.View())
	case wConfirm:
		b.WriteString(m.summary())
	}
	b.WriteString("\n\n" + helpStyle.Render("[↑/↓] choose  [enter] next  [esc] back  [ctrl+c] quit"))
	return b.String()
}

func (m wizardModel) summary() string {
	lines := []string{
		"Review:",
		"  engine:   " + string(m.cfg.Engine),
		"  timezone: " + m.cfg.Timezone,
	}
	if m.cfg.Schedule.Always {
		lines = append(lines, "  schedule: always on")
	} else if !m.cfg.Schedule.Enabled {
		lines = append(lines, "  schedule: manual only")
	} else {
		for _, w := range m.cfg.Schedule.Windows {
			lines = append(lines, "  schedule: "+strings.Join(w.Days, ",")+" "+w.Start+"–"+w.End)
		}
	}
	if m.cfg.UsesGraph() {
		lines = append(lines, "  graph:    "+emptyDash(m.cfg.Graph.ClientID)+" @ "+m.cfg.Graph.TenantID)
	}
	if m.flash != "" {
		lines = append(lines, "", flashStyle.Render(m.flash))
	}
	lines = append(lines, "", "Press [enter] to save.")
	return boxStyle.Render(strings.Join(lines, "\n"))
}

func renderChoices(title string, labels []string, cursor int) string {
	var b strings.Builder
	b.WriteString(title + "\n\n")
	for i, l := range labels {
		if i == cursor {
			b.WriteString(selRowStyle.Render("▸ "+l) + "\n")
		} else {
			b.WriteString("  " + l + "\n")
		}
	}
	return b.String()
}

func engineLabels() []string {
	return []string{
		"input  — synthetic activity (no account, default)",
		"graph  — Microsoft Graph presence (needs Entra app + admin consent)",
		"both   — run input and graph together",
	}
}

func presetLabels() []string {
	out := make([]string, len(presetChoices))
	for i, p := range presetChoices {
		out[i] = p.label
	}
	return out
}
