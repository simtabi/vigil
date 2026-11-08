// Package brand is the single source of vigil's identity (name, description,
// author, URLs) and its terminal presentation (the eye mark and the masthead
// banner). It is imported by both the CLI and the TUI.
package brand

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Identity — keep these exact; they appear in the banner, docs, and metadata.
const (
	Name = "vigil"
	// Pretty is the short product name (compact UI, e.g. the TUI header).
	Pretty = "Vigil"
	// Title is the full banner headline; it names the supported tools (more are
	// planned beyond Teams and Slack).
	Title       = "Vigil for MS Teams and Slack"
	Tagline     = "Keep Microsoft Teams active on a schedule."
	Description = "Keeps your Microsoft Teams presence green — on a schedule or " +
		"at will — using synthetic input or the Graph presence API."
	Author     = "Imani Manyara <imani@simtabi.com>"
	Contact    = "opensource@simtabi.com"
	URLProduct = "https://opensource.simtabi.com/products/vigil"
	URLRepo    = "https://github.com/simtabi/vigil"
)

// Eye is the brand mark in glyph form (an open eye with a presence pupil).
const Eye = "◉"

var (
	accentStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	boxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).Padding(0, 2)
)

// Banner renders the masthead shown by `vigil version` and the TUI onboarding.
// When color is false (NO_COLOR / not a TTY) it emits a plain ASCII box with no
// ANSI styling, safe for logs and pipes.
func Banner(version, commit, date string, color bool) string {
	rows := [][2]string{
		{"version", fmt.Sprintf("%s  (commit %s, built %s)", version, commit, date)},
		{"author", Author},
		{"contact", Contact},
		{"product", URLProduct},
		{"source", URLRepo},
	}
	if !color {
		lines := []string{
			Title + " — " + Tagline,
			"",
		}
		for _, r := range rows {
			lines = append(lines, fmt.Sprintf("%-8s %s", r[0], r[1]))
		}
		return asciiBox(lines)
	}

	var b strings.Builder
	b.WriteString(accentStyle.Render(Eye+" "+Title) + "  " + dimStyle.Render(Tagline))
	b.WriteString("\n")
	for _, r := range rows {
		b.WriteString("\n" + labelStyle.Render(fmt.Sprintf("%-8s", r[0])) + " " + r[1])
	}
	return boxStyle.Render(b.String())
}

// asciiBox draws a plain +/-/| box around left-aligned lines.
func asciiBox(lines []string) string {
	width := 0
	for _, l := range lines {
		if n := len(l); n > width {
			width = n
		}
	}
	rule := "+" + strings.Repeat("-", width+2) + "+"
	var b strings.Builder
	b.WriteString(rule + "\n")
	for _, l := range lines {
		b.WriteString(fmt.Sprintf("| %-*s |\n", width, l))
	}
	b.WriteString(rule)
	return b.String()
}
