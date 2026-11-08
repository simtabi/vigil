package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/simtabi/vigil/internal/tz"
)

// pickerWindow is how many list rows are visible at once (the list scrolls).
const pickerWindow = 10

// picker is a reusable searchable list: a filter input over a windowed,
// scrollable list of timezones. It is embedded by the settings editor and the
// setup wizard so timezone selection is identical in both.
type picker struct {
	title   string
	input   textinput.Model
	matches []string
	cursor  int
	top     int // index of the first visible row
}

func newPicker(title, current string) picker {
	ti := textinput.New()
	ti.Prompt = "search ▸ "
	ti.Placeholder = "type to filter… (e.g. new york, london, utc)"
	ti.CharLimit = 64
	ti.Focus()
	p := picker{title: title, input: ti, matches: tz.Filter("")}
	if i := indexOf(p.matches, current); i >= 0 {
		p.cursor = i
	}
	p.clamp()
	return p
}

// update processes one key. Return semantics:
//   - chosen != "" → a value was selected (commit it)
//   - cancelled == true → user pressed Esc (abandon)
//   - both zero → still picking
func (p *picker) update(key tea.KeyMsg) (chosen string, cancelled bool) {
	switch key.String() {
	case "esc":
		return "", true
	case "enter":
		if len(p.matches) > 0 {
			return p.matches[p.cursor], false
		}
		return "", false
	case "up", "ctrl+p":
		if p.cursor > 0 {
			p.cursor--
		}
		p.clamp()
		return "", false
	case "down", "ctrl+n":
		if p.cursor < len(p.matches)-1 {
			p.cursor++
		}
		p.clamp()
		return "", false
	}
	prev := p.input.Value()
	p.input, _ = p.input.Update(key)
	if p.input.Value() != prev {
		p.matches = tz.Filter(p.input.Value())
		p.cursor, p.top = 0, 0
	}
	return "", false
}

func (p *picker) clamp() {
	if p.cursor < p.top {
		p.top = p.cursor
	}
	if p.cursor >= p.top+pickerWindow {
		p.top = p.cursor - pickerWindow + 1
	}
	if p.top < 0 {
		p.top = 0
	}
}

func (p picker) view() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("  "+p.title) + "\n\n")
	b.WriteString("  " + p.input.View() + "\n\n")
	if len(p.matches) == 0 {
		b.WriteString(helpStyle.Render("  no matches"))
		return b.String()
	}
	end := p.top + pickerWindow
	if end > len(p.matches) {
		end = len(p.matches)
	}
	for i := p.top; i < end; i++ {
		if i == p.cursor {
			b.WriteString(selRowStyle.Render(" ▸ "+p.matches[i]) + "\n")
		} else {
			b.WriteString("   " + p.matches[i] + "\n")
		}
	}
	b.WriteString("\n" + helpStyle.Render(fmt.Sprintf("  %d match(es) · ↑/↓ move · type to filter · enter select · esc cancel", len(p.matches))))
	return b.String()
}

func indexOf(ss []string, want string) int {
	for i, s := range ss {
		if s == want {
			return i
		}
	}
	return -1
}
