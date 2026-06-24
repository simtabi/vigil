package tui

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/simtabi/ms-teams-activity/internal/config"
	"github.com/simtabi/ms-teams-activity/internal/control"
	"github.com/simtabi/ms-teams-activity/internal/schedule"
	"github.com/simtabi/ms-teams-activity/internal/selfupdate"
)

// --- helpers ---

func testOpts(t *testing.T, withConfig bool) Options {
	t.Helper()
	dir := t.TempDir()
	opts := Options{
		Scope:      config.ScopeUser,
		ConfigPath: filepath.Join(dir, "config.json"),
		RuntimeDir: dir,
		Version:    "0.0.0-dev",
	}
	if withConfig {
		if err := config.Default().Save(opts.ConfigPath); err != nil {
			t.Fatalf("save default config: %v", err)
		}
	}
	return opts
}

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

// press sends a key and returns the updated model (discarding the cmd).
func press(m model, s string) model {
	nm, _ := m.Update(keyMsg(s))
	return nm.(model)
}

// pressCmd sends a key and returns the updated model and the cmd.
func pressCmd(m model, s string) (model, tea.Cmd) {
	nm, cmd := m.Update(keyMsg(s))
	return nm.(model), cmd
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func cursorAt(m model, id menuID) model {
	for i, e := range m.menu {
		if e.id == id {
			m.cursor = i
		}
	}
	return m
}

// --- onboarding ---

func TestOnboarding(t *testing.T) {
	opts := testOpts(t, false)
	m := newModel(opts)
	if m.screen != screenOnboard {
		t.Fatalf("no config should start on onboard, got %v", m.screen)
	}
	// 'i' writes a default config and goes to the menu.
	m = press(m, "i")
	if m.screen != screenMenu {
		t.Fatalf("after init expected menu, got %v", m.screen)
	}
	if _, err := config.Load(opts.ConfigPath); err != nil {
		t.Fatalf("config not written: %v", err)
	}

	// 'w' returns a (non-nil) wizard exec cmd.
	m2 := newModel(testOpts(t, false))
	if _, cmd := pressCmd(m2, "w"); cmd == nil {
		t.Fatal("wizard key should return a command")
	}

	// 'q' quits.
	if _, cmd := pressCmd(newModel(testOpts(t, false)), "q"); !isQuit(cmd) {
		t.Fatal("q on onboard should quit")
	}
}

// --- menu navigation ---

func TestMenuNavigationClamps(t *testing.T) {
	m := newModel(testOpts(t, true))
	if m.screen != screenMenu {
		t.Fatalf("with config should start on menu, got %v", m.screen)
	}
	if m.cursor != 0 {
		t.Fatalf("cursor should start at 0")
	}
	m = press(m, "up") // clamp at 0
	if m.cursor != 0 {
		t.Fatalf("up at top should clamp to 0, got %d", m.cursor)
	}
	for i := 0; i < len(m.menu)+5; i++ {
		m = press(m, "down")
	}
	if m.cursor != len(m.menu)-1 {
		t.Fatalf("down should clamp to last (%d), got %d", len(m.menu)-1, m.cursor)
	}
	// j/k aliases
	m = press(m, "k")
	if m.cursor != len(m.menu)-2 {
		t.Fatalf("k should move up one, got %d", m.cursor)
	}
}

func TestMenuSelectTransitions(t *testing.T) {
	cases := []struct {
		id   menuID
		want screen
	}{
		{miStatus, screenDashboard},
		{miOverride, screenOverride},
		{miSchedule, screenSchedule},
		{miSettings, screenSettings},
		{miService, screenService},
		{miAccount, screenAccount},
		{miHelp, screenHelp},
	}
	for _, tc := range cases {
		m := cursorAt(newModel(testOpts(t, true)), tc.id)
		m = press(m, "enter")
		if m.screen != tc.want {
			t.Errorf("select %v → screen %v, want %v", tc.id, m.screen, tc.want)
		}
	}

	// Quit.
	if _, cmd := pressCmd(cursorAt(newModel(testOpts(t, true)), miQuit), "enter"); !isQuit(cmd) {
		t.Error("selecting Quit should quit")
	}

	// Update on a dev build flashes and stays on the menu.
	m := cursorAt(newModel(testOpts(t, true)), miUpdate)
	m, cmd := pressCmd(m, "enter")
	if cmd != nil || m.screen != screenMenu || m.flash == "" {
		t.Errorf("dev update should flash and stay on menu (cmd=%v screen=%v flash=%q)", cmd, m.screen, m.flash)
	}
}

func TestEscReturnsToMenu(t *testing.T) {
	for _, s := range []screen{screenDashboard, screenOverride, screenService, screenAccount, screenHelp} {
		m := newModel(testOpts(t, true))
		m.screen = s
		m = press(m, "esc")
		if m.screen != screenMenu {
			t.Errorf("esc from %v should return to menu, got %v", s, m.screen)
		}
	}
}

// --- override ---

func TestOverrideActions(t *testing.T) {
	m := newModel(testOpts(t, true))
	ovrPath := control.OverridePath(m.opts.RuntimeDir)

	// Force active.
	m.screen = screenOverride
	m.subCursor = 0 // "on"
	m = press(m, "enter")
	if m.screen != screenMenu {
		t.Fatal("override action should return to menu")
	}
	ov, _ := schedule.LoadOverride(ovrPath)
	if ov.Mode != schedule.OverrideOn {
		t.Fatalf("expected override on, got %q", ov.Mode)
	}

	// Force inactive.
	m.screen = screenOverride
	m.subCursor = 1 // "off"
	m = press(m, "enter")
	ov, _ = schedule.LoadOverride(ovrPath)
	if ov.Mode != schedule.OverrideOff {
		t.Fatalf("expected override off, got %q", ov.Mode)
	}

	// Resume clears it.
	m.screen = screenOverride
	m.subCursor = 2 // "resume"
	m = press(m, "enter")
	ov, _ = schedule.LoadOverride(ovrPath)
	if ov.Mode != schedule.OverrideNone {
		t.Fatalf("expected override cleared, got %q", ov.Mode)
	}
}

// --- schedule editor ---

func TestScheduleEditorAddAndSave(t *testing.T) {
	opts := testOpts(t, true)
	m := cursorAt(newModel(opts), miSchedule)
	m = press(m, "enter") // enter editor
	if m.screen != screenSchedule {
		t.Fatalf("should be on schedule screen, got %v", m.screen)
	}
	before := len(m.edit.Schedule.Windows)
	m = press(m, "a") // add a window
	if len(m.edit.Schedule.Windows) != before+1 {
		t.Fatalf("add window: got %d want %d", len(m.edit.Schedule.Windows), before+1)
	}
	m = press(m, "s") // save (default windows are valid)
	if m.screen != screenMenu {
		t.Fatalf("save should return to menu, got %v (flash %q)", m.screen, m.flash)
	}
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(cfg.Schedule.Windows) != before+1 {
		t.Fatalf("persisted windows: got %d want %d", len(cfg.Schedule.Windows), before+1)
	}
}

func TestScheduleEditorCancelDiscards(t *testing.T) {
	opts := testOpts(t, true)
	m := cursorAt(newModel(opts), miSchedule)
	m = press(m, "enter")
	m = press(m, "a") // add window in the working copy
	m = press(m, "esc")
	if m.screen != screenMenu {
		t.Fatalf("esc should return to menu")
	}
	cfg, _ := config.Load(opts.ConfigPath)
	if len(cfg.Schedule.Windows) != 1 {
		t.Fatalf("cancel must not persist; got %d windows", len(cfg.Schedule.Windows))
	}
}

func TestScheduleEditorTimeAndDayEdits(t *testing.T) {
	m := cursorAt(newModel(testOpts(t, true)), miSchedule)
	m = press(m, "enter")
	// field cycles days -> start -> end -> days
	if m.field != fieldDays {
		t.Fatal("starts on days field")
	}
	m = press(m, "tab")
	if m.field != fieldStart {
		t.Fatalf("tab should move to start field, got %d", m.field)
	}
	start := m.edit.Schedule.Windows[0].Start
	m = press(m, "+") // +15m
	if m.edit.Schedule.Windows[0].Start == start {
		t.Fatal("'+' should change the start time")
	}
	// toggle a day on the days field
	m.field = fieldDays
	before := len(m.edit.Schedule.Windows[0].Days)
	m = press(m, "6") // toggle Saturday (index 5)
	if len(m.edit.Schedule.Windows[0].Days) != before+1 {
		t.Fatalf("toggling a day should add it: got %d want %d", len(m.edit.Schedule.Windows[0].Days), before+1)
	}
}

// --- settings editor ---

func TestSettingsCycleAndClamp(t *testing.T) {
	m := cursorAt(newModel(testOpts(t, true)), miSettings)
	m = press(m, "enter")
	if m.screen != screenSettings {
		t.Fatalf("should be on settings, got %v", m.screen)
	}
	// engine cycle: input -> both
	m.setRow = rowEngine
	m = press(m, "space")
	if m.edit.Engine != config.EngineBoth {
		t.Fatalf("engine cycle: got %v want both", m.edit.Engine)
	}
	// interval clamp at lower bound
	m.setRow = rowInterval
	for i := 0; i < 50; i++ {
		m = press(m, "-")
	}
	if m.edit.Input.IntervalSeconds < 5 {
		t.Fatalf("interval should clamp >= 5, got %d", m.edit.Input.IntervalSeconds)
	}
	// prevent_sleep toggle
	m.setRow = rowPreventSleep
	ps := m.edit.Input.PreventSleep
	m = press(m, "space")
	if m.edit.Input.PreventSleep == ps {
		t.Fatal("prevent_sleep should toggle")
	}
}

func TestSettingsSaveInvalidGraphStays(t *testing.T) {
	m := cursorAt(newModel(testOpts(t, true)), miSettings)
	m = press(m, "enter")
	// cycle engine input -> both -> graph
	m.setRow = rowEngine
	m = press(m, "space") // both
	m = press(m, "space") // graph
	if m.edit.Engine != config.EngineGraph {
		t.Fatalf("expected graph engine, got %v", m.edit.Engine)
	}
	// save with empty client_id must fail and stay on settings
	m = press(m, "s")
	if m.screen != screenSettings {
		t.Fatalf("invalid save should stay on settings, got %v", m.screen)
	}
	if m.flash == "" {
		t.Fatal("invalid save should set a flash error")
	}
}

func TestSettingsTextEditFlow(t *testing.T) {
	m := cursorAt(newModel(testOpts(t, true)), miSettings)
	m = press(m, "enter")
	m.setRow = rowTimezone
	m = press(m, "enter") // begin editing
	if !m.setEditing {
		t.Fatal("enter on a text row should start editing")
	}
	m = press(m, "X") // type
	m = press(m, "enter")
	if m.setEditing {
		t.Fatal("enter should commit and stop editing")
	}
	if m.edit.Timezone == "Local" {
		t.Fatalf("timezone should have changed from edit, got %q", m.edit.Timezone)
	}
	// esc during editing cancels editing (not the screen)
	m.setRow = rowClientID
	m = press(m, "enter")
	m = press(m, "esc")
	if m.setEditing {
		t.Fatal("esc should stop editing")
	}
	if m.screen != screenSettings {
		t.Fatal("esc during edit should not leave settings")
	}
}

// --- service ---

func TestServiceSubmenu(t *testing.T) {
	m := newModel(testOpts(t, true))
	m.screen = screenService
	m.subCursor = 0 // "install"
	m, cmd := pressCmd(m, "enter")
	if cmd == nil {
		t.Fatal("service action should return an exec command")
	}
	if m.screen != screenMenu {
		t.Fatal("service action should return to menu")
	}
	// "Back" (last item) returns no command.
	m.screen = screenService
	m.subCursor = len(serviceActions) - 1
	m, cmd = pressCmd(m, "enter")
	if cmd != nil {
		t.Fatal("Back should not run a command")
	}
	if m.screen != screenMenu {
		t.Fatal("Back should return to menu")
	}
}

// --- non-key messages + view smoke ---

func TestNonKeyMessages(t *testing.T) {
	m := newModel(testOpts(t, true))
	// WindowSize is a no-op.
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if nm.(model).screen != screenMenu {
		t.Fatal("window size should not change screen")
	}
	// tick refreshes but keeps the screen.
	nm, cmd := m.Update(tickMsg{})
	if nm.(model).screen != screenMenu || cmd == nil {
		t.Fatal("tick should keep screen and reschedule")
	}
	// updateMsg sets the banner.
	nm2, _ := m.Update(updateMsg{info: selfupdate.Info{Available: true, Current: "1.0.0", Latest: "1.1.0"}})
	if !nm2.(model).update.Available {
		t.Fatal("updateMsg should set update info")
	}
}

func TestViewSmoke(t *testing.T) {
	screens := []screen{
		screenMenu, screenDashboard, screenOverride, screenSchedule,
		screenSettings, screenService, screenAccount, screenHelp, screenOnboard,
	}
	for _, s := range screens {
		m := newModel(testOpts(t, true))
		// schedule/settings views need their working copy loaded.
		m.enterEditor()
		m.enterSettings()
		m.screen = s
		if got := m.View(); got == "" {
			t.Errorf("View() empty for screen %v", s)
		}
	}
	// Error states render too.
	m := newModel(testOpts(t, false)) // no config → cfgErr/stErr set
	for _, s := range []screen{screenMenu, screenDashboard, screenOnboard} {
		m.screen = s
		if m.View() == "" {
			t.Errorf("View() empty for error-state screen %v", s)
		}
	}
}
