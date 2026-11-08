package tui

import tea "github.com/charmbracelet/bubbletea"

var serviceActions = []struct{ cmd, label, desc string }{
	{"install", "Install", "Install + start the background service"},
	{"start", "Start", "Start the installed service"},
	{"stop", "Stop", "Stop the service"},
	{"restart", "Restart", "Restart the service"},
	{"uninstall", "Uninstall", "Stop + remove the service"},
	{"", "Back", ""},
}

func (m model) updateService(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenMenu
	case "up", "k":
		m.subCursor = moveCursor(m.subCursor, len(serviceActions), -1)
	case "down", "j":
		m.subCursor = moveCursor(m.subCursor, len(serviceActions), +1)
	case "enter", "right", "l":
		a := serviceActions[m.subCursor]
		m.screen = screenMenu
		if a.cmd == "" {
			return m, nil
		}
		m.flash = "ran service " + a.cmd
		return m, m.execSelf(a.cmd)
	}
	return m, nil
}

func (m model) serviceBody() string {
	labels := make([]string, len(serviceActions))
	for i, a := range serviceActions {
		labels[i] = a.label
		if a.desc != "" {
			labels[i] += "  " + helpStyle.Render(a.desc)
		}
	}
	return listView(labels, m.subCursor)
}
