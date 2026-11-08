package schedule

import (
	"testing"
	"time"

	"github.com/simtabi/vigil/internal/config"
)

func cfgWith(tz string, always bool, win ...config.Window) config.Config {
	c := config.Default()
	c.Timezone = tz
	c.Schedule.Enabled = true
	c.Schedule.Always = always
	c.Schedule.Windows = win
	return c
}

func mustTime(t *testing.T, tz, value string) time.Time {
	t.Helper()
	loc, err := time.LoadLocation(tz)
	if err != nil {
		t.Fatalf("load location %q: %v", tz, err)
	}
	ts, err := time.ParseInLocation("2006-01-02 15:04", value, loc)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return ts
}

func TestActive_NormalWindow(t *testing.T) {
	tz := "America/New_York"
	cfg := cfgWith(tz, false, config.Window{
		Days: []string{"Mon", "Tue", "Wed", "Thu", "Fri"}, Start: "08:00", End: "17:00",
	})
	cases := []struct {
		name string
		when string // "2024-01-01 09:00" is a Monday
		want bool
	}{
		{"monday morning in window", "2024-01-01 09:00", true},
		{"monday at start boundary", "2024-01-01 08:00", true},
		{"monday at end boundary exclusive", "2024-01-01 17:00", false},
		{"monday before window", "2024-01-01 07:59", false},
		{"saturday not a work day", "2024-01-06 09:00", false},
		{"sunday not a work day", "2024-01-07 12:00", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Active(cfg, mustTime(t, tz, tc.when))
			if got != tc.want {
				t.Fatalf("Active(%s) = %v, want %v", tc.when, got, tc.want)
			}
		})
	}
}

func TestActive_OvernightWindow(t *testing.T) {
	tz := "UTC"
	// Friday 22:00 -> Saturday 06:00 night shift.
	cfg := cfgWith(tz, false, config.Window{
		Days: []string{"Fri"}, Start: "22:00", End: "06:00",
	})
	cases := []struct {
		name string
		when string
		want bool
	}{
		{"friday evening in shift", "2024-01-05 23:00", true}, // Fri
		{"saturday early morning still in shift", "2024-01-06 05:00", true},
		{"saturday at end boundary", "2024-01-06 06:00", false},
		{"friday before shift", "2024-01-05 21:00", false},
		{"saturday evening not in shift", "2024-01-06 23:00", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Active(cfg, mustTime(t, tz, tc.when))
			if got != tc.want {
				t.Fatalf("Active(%s) = %v, want %v", tc.when, got, tc.want)
			}
		})
	}
}

func TestActive_AlwaysAndDisabled(t *testing.T) {
	always := cfgWith("UTC", true)
	if !Active(always, mustTime(t, "UTC", "2024-01-06 03:00")) {
		t.Fatal("always-on should be active at any time")
	}
	disabled := cfgWith("UTC", false, config.Window{Days: []string{"Mon"}, Start: "08:00", End: "17:00"})
	disabled.Schedule.Enabled = false
	if Active(disabled, mustTime(t, "UTC", "2024-01-01 09:00")) {
		t.Fatal("disabled schedule should never be active")
	}
}

func TestActive_DSTSpringForward(t *testing.T) {
	// US DST 2024 begins 2024-03-10 02:00 -> 03:00. A 08:00-17:00 window must
	// still evaluate correctly on that day.
	tz := "America/New_York"
	cfg := cfgWith(tz, false, config.Window{Days: []string{"Sun"}, Start: "08:00", End: "17:00"})
	if !Active(cfg, mustTime(t, tz, "2024-03-10 09:00")) {
		t.Fatal("expected active after spring-forward")
	}
}

func TestDesiredActive_OverridePrecedence(t *testing.T) {
	tz := "UTC"
	cfg := cfgWith(tz, false, config.Window{Days: []string{"Mon"}, Start: "08:00", End: "17:00"})
	out := mustTime(t, tz, "2024-01-06 12:00") // Saturday, schedule inactive
	in := mustTime(t, tz, "2024-01-01 12:00")  // Monday noon, schedule active

	if DesiredActive(cfg, Override{Mode: OverrideOn}, out) != true {
		t.Fatal("override on must force active outside schedule")
	}
	if DesiredActive(cfg, Override{Mode: OverrideOff}, in) != false {
		t.Fatal("override off must force inactive inside schedule")
	}
	if DesiredActive(cfg, Override{}, in) != true {
		t.Fatal("no override must follow schedule (active)")
	}
}

func TestOverride_Expiry(t *testing.T) {
	now := mustTime(t, "UTC", "2024-01-06 12:00")
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	expired := Override{Mode: OverrideOn, Until: &past}
	if expired.EffectiveMode(now) != OverrideNone {
		t.Fatal("expired override should be treated as none")
	}
	live := Override{Mode: OverrideOn, Until: &future}
	if live.EffectiveMode(now) != OverrideOn {
		t.Fatal("live override should remain in force")
	}
}

func TestNextChange(t *testing.T) {
	tz := "UTC"
	cfg := cfgWith(tz, false, config.Window{
		Days: []string{"Mon", "Tue", "Wed", "Thu", "Fri"}, Start: "08:00", End: "17:00",
	})
	// Monday 07:30 -> next change at 08:00 to active.
	when, toActive, ok := NextChange(cfg, Override{}, mustTime(t, tz, "2024-01-01 07:30"))
	if !ok || !toActive {
		t.Fatalf("expected upcoming activation, got ok=%v active=%v", ok, toActive)
	}
	if got := when.Format("15:04"); got != "08:00" {
		t.Fatalf("next change at %s, want 08:00", got)
	}
}
