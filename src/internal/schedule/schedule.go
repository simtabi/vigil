// Package schedule evaluates whether the daemon should be active at a given
// instant, combining the configured weekly windows with any manual override.
package schedule

import (
	"time"

	"github.com/simtabi/vigil/internal/config"
)

// weekdayAbbrev maps a time.Weekday to the 3-letter form used in config.
var weekdayAbbrev = map[time.Weekday]string{
	time.Monday:    "Mon",
	time.Tuesday:   "Tue",
	time.Wednesday: "Wed",
	time.Thursday:  "Thu",
	time.Friday:    "Fri",
	time.Saturday:  "Sat",
	time.Sunday:    "Sun",
}

// Active reports whether the configured schedule (ignoring overrides) is active
// at time t. The instant is evaluated in the config's timezone.
func Active(cfg config.Config, t time.Time) bool {
	if !cfg.Schedule.Enabled {
		return false
	}
	if cfg.Schedule.Always {
		return true
	}
	loc, err := cfg.Location()
	if err != nil {
		loc = time.Local
	}
	local := t.In(loc)
	mins := local.Hour()*60 + local.Minute()
	today := weekdayAbbrev[local.Weekday()]
	yesterday := weekdayAbbrev[local.AddDate(0, 0, -1).Weekday()]

	for _, w := range cfg.Schedule.Windows {
		start, err := config.ParseClock(w.Start)
		if err != nil {
			continue
		}
		end, err := config.ParseClock(w.End)
		if err != nil {
			continue
		}
		days := dayset(w.Days)
		if end.Minutes() > start.Minutes() {
			// Normal same-day window.
			if days[today] && mins >= start.Minutes() && mins < end.Minutes() {
				return true
			}
			continue
		}
		// Overnight window (end <= start): evening segment on a start day, and
		// morning segment that belongs to the day after a start day.
		if days[today] && mins >= start.Minutes() {
			return true
		}
		if days[yesterday] && mins < end.Minutes() {
			return true
		}
	}
	return false
}

// DesiredActive applies the override on top of the schedule and returns the
// effective desired state at time t. An expired override is ignored.
func DesiredActive(cfg config.Config, ov Override, t time.Time) bool {
	switch ov.EffectiveMode(t) {
	case OverrideOn:
		return true
	case OverrideOff:
		return false
	default:
		return Active(cfg, t)
	}
}

// NextChange scans forward from t (up to ~14 days) and returns the next instant
// at which DesiredActive flips, plus the state it flips to. ok is false if no
// change is found within the horizon (e.g. an indefinite override or 24/7).
func NextChange(cfg config.Config, ov Override, t time.Time) (when time.Time, toActive bool, ok bool) {
	current := DesiredActive(cfg, ov, t)
	// Step by one minute; cheap and exact at minute resolution. Align to the
	// next whole minute first.
	cursor := t.Truncate(time.Minute).Add(time.Minute)
	horizon := t.Add(14 * 24 * time.Hour)
	for cursor.Before(horizon) {
		if DesiredActive(cfg, ov, cursor) != current {
			return cursor, !current, true
		}
		cursor = cursor.Add(time.Minute)
	}
	return time.Time{}, false, false
}

func dayset(days []string) map[string]bool {
	m := make(map[string]bool, len(days))
	for _, d := range days {
		m[normalize(d)] = true
	}
	return m
}

func normalize(d string) string {
	if len(d) < 3 {
		return d
	}
	d = d[:3]
	return string(upper(d[0])) + lower(d[1:])
}

func upper(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - 32
	}
	return b
}

func lower(s string) string {
	out := []byte(s)
	for i, b := range out {
		if b >= 'A' && b <= 'Z' {
			out[i] = b + 32
		}
	}
	return string(out)
}
