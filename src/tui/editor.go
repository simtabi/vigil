package tui

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/simtabi/vigil/internal/config"
)

// dayOrder is the canonical weekday ordering used when rendering/toggling days.
var dayOrder = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

const (
	fieldDays = iota
	fieldStart
	fieldEnd
)

// enterEditor loads the current config into a working copy and switches modes.
func (m *model) enterEditor() {
	cfg, err := config.Load(m.opts.ConfigPath)
	if err != nil {
		m.flash = "cannot edit: " + err.Error()
		return
	}
	m.edit = cfg
	m.screen = screenSchedule
	m.winIdx = 0
	m.field = fieldDays
	m.flash = ""
}

func (m model) updateEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	wins := &m.edit.Schedule.Windows
	switch msg.String() {
	case "esc":
		m.screen = screenMenu
		m.flash = "edit cancelled"
	case "s":
		if err := m.edit.Validate(); err != nil {
			m.flash = "not saved: " + err.Error()
			break
		}
		if err := m.edit.Save(m.opts.ConfigPath); err != nil {
			m.flash = "save failed: " + err.Error()
			break
		}
		m.screen = screenMenu
		m.flash = "schedule saved"
		m.refresh()
	case "up", "k":
		if m.winIdx > 0 {
			m.winIdx--
		}
	case "down", "j":
		if m.winIdx < len(*wins)-1 {
			m.winIdx++
		}
	case "tab", "right", "l":
		m.field = (m.field + 1) % 3
	case "shift+tab", "left", "h":
		m.field = (m.field + 2) % 3
	case "t":
		m.edit.Schedule.Enabled = !m.edit.Schedule.Enabled
	case "y":
		m.edit.Schedule.Always = !m.edit.Schedule.Always
	case "a":
		*wins = append(*wins, config.Window{
			Days: []string{"Mon", "Tue", "Wed", "Thu", "Fri"}, Start: "09:00", End: "17:00",
		})
		m.winIdx = len(*wins) - 1
	case "d":
		if len(*wins) > 0 {
			*wins = slices.Delete(*wins, m.winIdx, m.winIdx+1)
			if m.winIdx >= len(*wins) && m.winIdx > 0 {
				m.winIdx--
			}
		}
	case "+", "=":
		m.adjustTime(15)
	case "-", "_":
		m.adjustTime(-15)
	default:
		// Number keys 1..7 toggle days when the days field is focused.
		if m.field == fieldDays && len(msg.String()) == 1 {
			if d := msg.String()[0]; d >= '1' && d <= '7' {
				m.toggleDay(int(d - '1'))
			}
		}
	}
	return m, nil
}

// adjustTime shifts the focused start/end field by delta minutes (wrapping).
func (m *model) adjustTime(delta int) {
	wins := m.edit.Schedule.Windows
	if m.winIdx >= len(wins) || m.field == fieldDays {
		return
	}
	w := &wins[m.winIdx]
	target := &w.Start
	if m.field == fieldEnd {
		target = &w.End
	}
	c, err := config.ParseClock(*target)
	if err != nil {
		return
	}
	mins := (c.Minutes() + delta + 1440) % 1440
	*target = config.Clock(mins).String()
}

// toggleDay adds/removes the day at dayOrder[idx] from the selected window,
// keeping days in canonical order.
func (m *model) toggleDay(idx int) {
	wins := m.edit.Schedule.Windows
	if m.winIdx >= len(wins) || idx < 0 || idx >= len(dayOrder) {
		return
	}
	day := dayOrder[idx]
	w := &wins[m.winIdx]
	if slices.Contains(w.Days, day) {
		w.Days = slices.DeleteFunc(w.Days, func(d string) bool { return d == day })
		return
	}
	w.Days = append(w.Days, day)
	slices.SortFunc(w.Days, func(a, b string) int {
		return slices.Index(dayOrder, a) - slices.Index(dayOrder, b)
	})
}

func (m model) editorView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("  Edit schedule") + "  " + helpStyle.Render("(unsaved — press s to save)") + "\n\n")

	flags := fmt.Sprintf("enabled: %v    always: %v    tz: %s    engine: %s",
		m.edit.Schedule.Enabled, m.edit.Schedule.Always, m.edit.Timezone, m.edit.Engine)
	b.WriteString(boxStyle.Render(labelStyle.Render(flags)) + "\n")

	var rows []string
	if len(m.edit.Schedule.Windows) == 0 {
		rows = append(rows, labelStyle.Render("  (no windows — press 'a' to add one)"))
	}
	for i, w := range m.edit.Schedule.Windows {
		marker := "  "
		if i == m.winIdx {
			marker = selRowStyle.Render("▸ ")
		}
		rows = append(rows, marker+m.renderWindow(i, w))
	}
	b.WriteString(boxStyle.Render(strings.Join(rows, "\n")) + "\n")

	if m.flash != "" {
		b.WriteString(flashStyle.Render("• "+m.flash) + "\n")
	}
	b.WriteString(helpStyle.Render("[↑/↓] window  [tab] field  [1-7] toggle day  [+/-] time  [a]dd  [d]el  [t]oggle enabled  [y] always  [s]ave  [esc] cancel"))
	return b.String()
}

func (m model) renderWindow(i int, w config.Window) string {
	selected := i == m.winIdx
	// Days as M T W T F S S, highlighting active ones.
	var days []string
	for _, d := range dayOrder {
		st := dayOffStyle
		if slices.Contains(w.Days, d) {
			st = dayOnStyle
		}
		days = append(days, st.Render(d[:1]))
	}
	daysStr := strings.Join(days, " ")
	if selected && m.field == fieldDays {
		daysStr = selFieldStyle.Render(" " + strings.Join(daysFor(w), " ") + " ")
	}

	start := w.Start
	end := w.End
	if selected && m.field == fieldStart {
		start = selFieldStyle.Render(" " + w.Start + " ")
	}
	if selected && m.field == fieldEnd {
		end = selFieldStyle.Render(" " + w.End + " ")
	}
	return fmt.Sprintf("%s   %s–%s", daysStr, start, end)
}

// daysFor renders the M/T/W markers as plain letters for the highlighted field.
func daysFor(w config.Window) []string {
	out := make([]string, len(dayOrder))
	for i, d := range dayOrder {
		if slices.Contains(w.Days, d) {
			out[i] = d[:1]
		} else {
			out[i] = "·"
		}
	}
	return out
}
