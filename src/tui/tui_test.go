package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/simtabi/vigil/internal/config"
	"github.com/simtabi/vigil/internal/control"
	"github.com/simtabi/vigil/internal/schedule"
	"github.com/simtabi/vigil/internal/selfupdate"
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

func onboardIndex(key string) int {
	for i, it := range onboardItems {
		if it.key == key {
			return i
		}
	}
	return -1
}

func TestOnboarding(t *testing.T) {
	opts := testOpts(t, false)
	m := newModel(opts)
	if m.screen != screenOnboard {
		t.Fatalf("no config should start on onboard, got %v", m.screen)
	}
	// Onboarding is a navigable menu: cursor clamps at the top.
	m = press(m, "up")
	if m.subCursor != 0 {
		t.Fatalf("onboard cursor should clamp at 0, got %d", m.subCursor)
	}

	// Select "Write default config" → writes config and enters the menu.
	m.subCursor = onboardIndex("defaults")
	m = press(m, "enter")
	if m.screen != screenMenu {
		t.Fatalf("after default-config expected menu, got %v", m.screen)
	}
	if _, err := config.Load(opts.ConfigPath); err != nil {
		t.Fatalf("config not written: %v", err)
	}

	// "Guided setup wizard" returns a (non-nil) exec cmd.
	w := newModel(testOpts(t, false))
	w.subCursor = onboardIndex("wizard")
	if _, cmd := pressCmd(w, "enter"); cmd == nil {
		t.Fatal("wizard selection should return a command")
	}

	// Selecting "Quit" (and the q shortcut) both quit.
	q := newModel(testOpts(t, false))
	q.subCursor = onboardIndex("quit")
	if _, cmd := pressCmd(q, "enter"); !isQuit(cmd) {
		t.Fatal("selecting Quit should quit")
	}
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

func overrideIndex(key string, dur time.Duration) int {
	for i, it := range overrideItems {
		if it.key == key && it.dur == dur {
			return i
		}
	}
	return -1
}

func TestOverrideActions(t *testing.T) {
	m := newModel(testOpts(t, true))
	ovrPath := control.OverridePath(m.opts.RuntimeDir)

	pick := func(idx int) schedule.Override {
		m.screen = screenOverride
		m.subCursor = idx
		m = press(m, "enter")
		if m.screen != screenMenu {
			t.Fatalf("override action should return to menu")
		}
		ov, _ := schedule.LoadOverride(ovrPath)
		return ov
	}

	if ov := pick(overrideIndex("on", 0)); ov.Mode != schedule.OverrideOn || ov.Until != nil {
		t.Fatalf("indefinite on: got mode=%q until=%v", ov.Mode, ov.Until)
	}
	if ov := pick(overrideIndex("off", 0)); ov.Mode != schedule.OverrideOff {
		t.Fatalf("expected override off, got %q", ov.Mode)
	}
	if ov := pick(overrideIndex("resume", 0)); ov.Mode != schedule.OverrideNone {
		t.Fatalf("expected override cleared, got %q", ov.Mode)
	}
}

func TestOverrideTimed(t *testing.T) {
	m := newModel(testOpts(t, true))
	ovrPath := control.OverridePath(m.opts.RuntimeDir)
	idx := overrideIndex("on", 2*time.Hour)
	if idx < 0 {
		t.Fatal("expected a 2-hour override preset")
	}
	m.screen = screenOverride
	m.subCursor = idx
	before := time.Now()
	m = press(m, "enter")
	ov, _ := schedule.LoadOverride(ovrPath)
	if ov.Mode != schedule.OverrideOn || ov.Until == nil {
		t.Fatalf("timed override should set mode on + Until; got mode=%q until=%v", ov.Mode, ov.Until)
	}
	lo, hi := before.Add(2*time.Hour-time.Minute), time.Now().Add(2*time.Hour+time.Minute)
	if ov.Until.Before(lo) || ov.Until.After(hi) {
		t.Fatalf("Until %v not ~2h out", ov.Until)
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
	// jitter can never exceed interval-1 (config.Validate requires jitter < interval).
	m.setRow = rowInterval
	for i := 0; i < 50; i++ {
		m = press(m, "-")
	}
	m.setRow = rowJitter
	for i := 0; i < 100; i++ {
		m = press(m, "+")
	}
	if m.edit.Input.JitterSeconds >= m.edit.Input.IntervalSeconds {
		t.Fatalf("jitter %d must stay below interval %d", m.edit.Input.JitterSeconds, m.edit.Input.IntervalSeconds)
	}
}

func TestConnectivityIndicator(t *testing.T) {
	m := newModel(testOpts(t, true))
	if got := m.netStrip(); !strings.Contains(got, "net") {
		t.Fatalf("expected unknown net state before a check, got %q", got)
	}
	// Drive the async result deterministically (no live probe in tests).
	on, _ := m.Update(netMsg{online: true})
	if got := on.(model).netStrip(); !strings.Contains(got, "online") {
		t.Fatalf("expected online, got %q", got)
	}
	off, _ := m.Update(netMsg{online: false})
	if got := off.(model).netStrip(); !strings.Contains(got, "offline") {
		t.Fatalf("expected offline, got %q", got)
	}
}

func TestOverrideTimedOff(t *testing.T) {
	m := newModel(testOpts(t, true))
	ovrPath := control.OverridePath(m.opts.RuntimeDir)
	idx := overrideIndex("off", time.Hour)
	if idx < 0 {
		t.Fatal("expected a 1-hour force-inactive preset")
	}
	m.screen = screenOverride
	m.subCursor = idx
	m = press(m, "enter")
	ov, _ := schedule.LoadOverride(ovrPath)
	if ov.Mode != schedule.OverrideOff || ov.Until == nil {
		t.Fatalf("timed off should set mode off + Until; got mode=%q until=%v", ov.Mode, ov.Until)
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
	m.setRow = rowClientID // a free-text row (timezone uses the picker)
	m = press(m, "enter")  // begin editing
	if !m.setEditing {
		t.Fatal("enter on a text row should start editing")
	}
	m = press(m, "X") // type
	m = press(m, "enter")
	if m.setEditing {
		t.Fatal("enter should commit and stop editing")
	}
	if m.edit.Graph.ClientID == "" {
		t.Fatalf("client id should have changed from edit, got %q", m.edit.Graph.ClientID)
	}
	// esc during editing cancels editing (not the screen)
	m.setRow = rowTenantID
	m = press(m, "enter")
	m = press(m, "esc")
	if m.setEditing {
		t.Fatal("esc should stop editing")
	}
	if m.screen != screenSettings {
		t.Fatal("esc during edit should not leave settings")
	}
}

func TestTimezonePicker(t *testing.T) {
	m := cursorAt(newModel(testOpts(t, true)), miSettings)
	m = press(m, "enter") // enter settings
	m.setRow = rowTimezone
	m = press(m, "enter") // open the searchable picker
	if m.screen != screenTZ {
		t.Fatalf("timezone row should open the picker, got %v", m.screen)
	}
	// Type to filter, then select the top match.
	for _, k := range []string{"u", "t", "c"} {
		m = press(m, k)
	}
	if len(m.tz.matches) == 0 {
		t.Fatal("filtering 'utc' should yield matches")
	}
	m = press(m, "enter")
	if m.screen != screenSettings {
		t.Fatalf("selecting should return to settings, got %v", m.screen)
	}
	if !strings.Contains(strings.ToLower(m.edit.Timezone), "utc") {
		t.Fatalf("expected a UTC zone, got %q", m.edit.Timezone)
	}

	// Esc cancels back to settings without changing the value.
	m.setRow = rowTimezone
	before := m.edit.Timezone
	m = press(m, "enter")
	m = press(m, "esc")
	if m.screen != screenSettings {
		t.Fatal("esc should return to settings")
	}
	if m.edit.Timezone != before {
		t.Fatalf("esc should not change the timezone, got %q want %q", m.edit.Timezone, before)
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
