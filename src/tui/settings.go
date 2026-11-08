package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/simtabi/vigil/internal/config"
)

// settings rows
const (
	rowEngine = iota
	rowMethod
	rowInterval
	rowJitter
	rowPreventSleep
	rowTimezone
	rowClientID
	rowTenantID
	rowCount
)

var textRows = map[int]bool{rowTimezone: true, rowClientID: true, rowTenantID: true}

func (m *model) enterSettings() {
	cfg, err := config.Load(m.opts.ConfigPath)
	if err != nil {
		m.flash = "cannot edit settings: " + err.Error()
		return
	}
	m.edit = cfg
	m.screen = screenSettings
	m.setRow = 0
	m.setEditing = false
	m.flash = ""
}

func (m model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.setEditing {
		switch msg.String() {
		case "enter":
			m.commitTextRow(m.setInput.Value())
			m.setEditing = false
			m.setInput.Blur()
			return m, nil
		case "esc":
			m.setEditing = false
			m.setInput.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.setInput, cmd = m.setInput.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc":
		m.screen = screenMenu
		m.flash = "settings closed (unsaved changes discarded)"
	case "s":
		if err := m.edit.Validate(); err != nil {
			m.flash = "not saved: " + err.Error()
			return m, nil
		}
		if err := m.edit.Save(m.opts.ConfigPath); err != nil {
			m.flash = "save failed: " + err.Error()
			return m, nil
		}
		m.screen = screenMenu
		m.flash = "settings saved"
		m.refresh()
	case "up", "k":
		if m.setRow > 0 {
			m.setRow--
		}
	case "down", "j":
		if m.setRow < rowCount-1 {
			m.setRow++
		}
	case "left", "h", "-":
		m.adjustRow(-1)
	case "right", "l", "+", "=":
		m.adjustRow(1)
	case " ":
		m.adjustRow(1)
	case "enter":
		if m.setRow == rowTimezone {
			// Timezone uses the searchable picker rather than free-text entry.
			m.tz = newPicker("Timezone", m.edit.Timezone)
			m.screen = screenTZ
			return m, nil
		}
		if textRows[m.setRow] {
			m.setInput.SetValue(m.currentText())
			m.setInput.CursorEnd()
			m.setInput.Focus()
			m.setEditing = true
		}
	}
	return m, nil
}

func (m *model) adjustRow(dir int) {
	switch m.setRow {
	case rowEngine:
		m.edit.Engine = cycleEngine(m.edit.Engine, dir)
	case rowMethod:
		m.edit.Input.Method = cycleMethod(m.edit.Input.Method, dir)
	case rowInterval:
		m.edit.Input.IntervalSeconds = clampInt(m.edit.Input.IntervalSeconds+dir*5, 5, 299)
		// Jitter must stay below the interval (config.Validate enforces jitter < interval).
		m.edit.Input.JitterSeconds = clampInt(m.edit.Input.JitterSeconds, 0, m.edit.Input.IntervalSeconds-1)
	case rowJitter:
		m.edit.Input.JitterSeconds = clampInt(m.edit.Input.JitterSeconds+dir*5, 0, m.edit.Input.IntervalSeconds-1)
	case rowPreventSleep:
		m.edit.Input.PreventSleep = !m.edit.Input.PreventSleep
	}
}

func (m model) currentText() string {
	switch m.setRow {
	case rowTimezone:
		return m.edit.Timezone
	case rowClientID:
		return m.edit.Graph.ClientID
	case rowTenantID:
		return m.edit.Graph.TenantID
	}
	return ""
}

func (m *model) commitTextRow(v string) {
	v = strings.TrimSpace(v)
	switch m.setRow {
	case rowTimezone:
		m.edit.Timezone = v
	case rowClientID:
		m.edit.Graph.ClientID = v
	case rowTenantID:
		m.edit.Graph.TenantID = v
	}
}

func (m model) settingsView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("  Settings") + "  " + helpStyle.Render("(unsaved — press s to save)") + "\n\n")

	rows := []struct{ label, val string }{
		{"engine", string(m.edit.Engine)},
		{"input.method", string(m.edit.Input.Method)},
		{"input.interval_seconds", fmt.Sprintf("%d", m.edit.Input.IntervalSeconds)},
		{"input.jitter_seconds", fmt.Sprintf("%d", m.edit.Input.JitterSeconds)},
		{"input.prevent_sleep", fmt.Sprintf("%v", m.edit.Input.PreventSleep)},
		{"timezone", m.edit.Timezone},
		{"graph.client_id", emptyDash(m.edit.Graph.ClientID)},
		{"graph.tenant_id", emptyDash(m.edit.Graph.TenantID)},
	}

	var lines []string
	for i, r := range rows {
		cursor := "  "
		label := labelStyle.Render(fmt.Sprintf("%-22s", r.label))
		val := r.val
		if i == m.setRow {
			cursor = selRowStyle.Render("▸ ")
			if m.setEditing && textRows[i] {
				val = m.setInput.View()
			} else {
				val = selFieldStyle.Render(" " + r.val + " ")
			}
		}
		lines = append(lines, cursor+label+val)
	}
	b.WriteString(boxStyle.Render(strings.Join(lines, "\n")) + "\n")
	if m.flash != "" {
		b.WriteString(flashStyle.Render("• "+m.flash) + "\n")
	}
	b.WriteString(helpStyle.Render("[↑/↓] row  [←/→/space] change  [enter] edit text  [s]ave  [esc] cancel"))
	return b.String()
}

func cycleEngine(e config.Engine, dir int) config.Engine {
	order := []config.Engine{config.EngineInput, config.EngineBoth, config.EngineGraph}
	return order[cycleIndex(indexEngine(order, e), len(order), dir)]
}

func cycleMethod(mth config.InputMethod, dir int) config.InputMethod {
	order := []config.InputMethod{config.MethodMouse, config.MethodKey, config.MethodZen}
	return order[cycleIndex(indexMethod(order, mth), len(order), dir)]
}

func indexEngine(s []config.Engine, e config.Engine) int {
	for i, v := range s {
		if v == e {
			return i
		}
	}
	return 0
}

func indexMethod(s []config.InputMethod, m config.InputMethod) int {
	for i, v := range s {
		if v == m {
			return i
		}
	}
	return 0
}

func cycleIndex(cur, n, dir int) int { return ((cur+dir)%n + n) % n }

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func emptyDash(s string) string {
	if s == "" {
		return "(unset)"
	}
	return s
}
