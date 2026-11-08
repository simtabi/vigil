// Package ui centralizes terminal feedback for the CLI: TTY/color detection,
// status lines, confirmation prompts, and spinners. Following CLI conventions,
// human-facing status goes to stderr and is gated on a real terminal; machine
// output (--json etc.) is the command's own responsibility on stdout.
package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"golang.org/x/term"
)

// Behavior toggles, set once from global flags by the root command.
var (
	assumeYes bool
	noInput   bool
	noColor   bool
)

// SetAssumeYes makes Confirm always return true (the `--yes` flag).
func SetAssumeYes(v bool) { assumeYes = v }

// SetNoInput makes Confirm never prompt and return the default (`--no-input`).
func SetNoInput(v bool) { noInput = v }

// SetNoColor disables ANSI color/icons (`--no-color`).
func SetNoColor(v bool) { noColor = v }

// AssumeYes reports whether `--yes` was set (for callers that branch on it).
func AssumeYes() bool { return assumeYes }

// IsTTY reports whether stdout is an interactive terminal.
func IsTTY() bool { return term.IsTerminal(int(os.Stdout.Fd())) }

func stdinTTY() bool { return term.IsTerminal(int(os.Stdin.Fd())) }

// ColorEnabled reports whether colored output should be produced. It honors
// `--no-color`, the NO_COLOR convention, TERM=dumb, and non-TTY stdout.
func ColorEnabled() bool {
	if noColor || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	return IsTTY()
}

func paint(code, s string) string {
	if !ColorEnabled() {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

// mark returns a colored unicode icon when color is on, else an ASCII tag.
func mark(unicode, ascii, code string) string {
	if ColorEnabled() {
		return paint(code, unicode)
	}
	return ascii
}

// Success/Fail/Warn/Info print a status line to stderr.
func Success(format string, a ...any) { line(mark("✓", "[OK]", "32"), format, a...) }
func Fail(format string, a ...any)    { line(mark("✗", "[FAIL]", "31"), format, a...) }
func Warn(format string, a ...any)    { line(mark("⚠", "[WARN]", "33"), format, a...) }
func Info(format string, a ...any)    { line(mark("i", "[i]", "36"), format, a...) }

func line(prefix, format string, a ...any) {
	fmt.Fprintln(os.Stderr, prefix+" "+fmt.Sprintf(format, a...))
}

// Confirm asks a yes/no question with a safe default. It returns true on `--yes`,
// the default when `--no-input` or stdin is not a terminal, and otherwise prompts
// (EOF/blank → default). The prompt and answer flow use stderr/stdin so stdout
// stays clean for machine output.
func Confirm(prompt string, def bool) bool {
	if assumeYes {
		return true
	}
	if noInput || !stdinTTY() {
		return def
	}
	suffix := "[y/N]"
	if def {
		suffix = "[Y/n]"
	}
	fmt.Fprintf(os.Stderr, "%s %s ", prompt, suffix)
	in, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(in)) {
	case "":
		return def
	case "y", "yes":
		return true
	default:
		return false
	}
}

// Spin runs fn while showing a spinner (on a TTY) or a plain "…" line (otherwise).
// It returns fn's error; callers report success/failure themselves.
func Spin(title string, fn func() error) error {
	if !IsTTY() {
		fmt.Fprintln(os.Stderr, title+"…")
		return fn()
	}
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(io.Writer(os.Stderr)))
	s.Suffix = " " + title
	s.Start()
	defer s.Stop()
	return fn()
}
