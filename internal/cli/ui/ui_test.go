package ui

import "testing"

func reset() {
	assumeYes, noInput, noColor = false, false, false
}

func TestConfirmAssumeYes(t *testing.T) {
	t.Cleanup(reset)
	SetAssumeYes(true)
	if !Confirm("proceed?", false) {
		t.Fatal("--yes should force true even with a false default")
	}
}

func TestConfirmNoInputReturnsDefault(t *testing.T) {
	t.Cleanup(reset)
	SetNoInput(true)
	if !Confirm("proceed?", true) {
		t.Fatal("--no-input should return the (true) default")
	}
	if Confirm("proceed?", false) {
		t.Fatal("--no-input should return the (false) default")
	}
}

func TestConfirmNonInteractiveReturnsDefault(t *testing.T) {
	t.Cleanup(reset)
	// In `go test`, stdin is not a terminal, so Confirm must not block and must
	// return the default rather than reading.
	if !Confirm("proceed?", true) {
		t.Fatal("non-interactive Confirm should return the true default")
	}
	if Confirm("proceed?", false) {
		t.Fatal("non-interactive Confirm should return the false default")
	}
}

func TestColorDisabled(t *testing.T) {
	t.Cleanup(reset)
	SetNoColor(true)
	if ColorEnabled() {
		t.Fatal("--no-color should disable color")
	}
	reset()
	t.Setenv("NO_COLOR", "1")
	if ColorEnabled() {
		t.Fatal("NO_COLOR should disable color")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if ColorEnabled() {
		t.Fatal("TERM=dumb should disable color")
	}
}
