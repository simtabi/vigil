package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Clock is a wall-clock time of day expressed as minutes since midnight,
// in the range [0,1440).
type Clock int

// ParseClock parses an "HH:MM" 24-hour string into a Clock.
func ParseClock(s string) (Clock, error) {
	s = strings.TrimSpace(s)
	h, m, ok := strings.Cut(s, ":")
	if !ok {
		return 0, fmt.Errorf("expected HH:MM, got %q", s)
	}
	hh, err := strconv.Atoi(h)
	if err != nil || hh < 0 || hh > 23 {
		return 0, fmt.Errorf("invalid hour in %q", s)
	}
	mm, err := strconv.Atoi(m)
	if err != nil || mm < 0 || mm > 59 {
		return 0, fmt.Errorf("invalid minute in %q", s)
	}
	return Clock(hh*60 + mm), nil
}

// Minutes returns the clock value as minutes since midnight.
func (c Clock) Minutes() int { return int(c) }

// String renders the clock as "HH:MM".
func (c Clock) String() string { return fmt.Sprintf("%02d:%02d", int(c)/60, int(c)%60) }

// ParseISODuration parses the subset of ISO-8601 durations used by Microsoft
// Graph (days, hours, minutes, seconds — e.g. "PT8H", "P1DT2H30M"). Week, month
// and year designators are rejected because they are ambiguous as fixed
// durations.
func ParseISODuration(s string) (time.Duration, error) {
	orig := s
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	if !strings.HasPrefix(s, "P") {
		return 0, fmt.Errorf("invalid ISO-8601 duration %q (must start with P)", orig)
	}
	s = s[1:]
	datePart, timePart, hasT := strings.Cut(s, "T")

	var total time.Duration
	parse := func(part string, units map[byte]time.Duration) error {
		num := strings.Builder{}
		for i := 0; i < len(part); i++ {
			ch := part[i]
			if (ch >= '0' && ch <= '9') || ch == '.' {
				num.WriteByte(ch)
				continue
			}
			unit, ok := units[ch]
			if !ok {
				return fmt.Errorf("unsupported designator %q in %q", string(ch), orig)
			}
			if num.Len() == 0 {
				return fmt.Errorf("missing value before %q in %q", string(ch), orig)
			}
			v, err := strconv.ParseFloat(num.String(), 64)
			if err != nil {
				return fmt.Errorf("invalid number in %q: %w", orig, err)
			}
			total += time.Duration(v * float64(unit))
			num.Reset()
		}
		if num.Len() != 0 {
			return fmt.Errorf("trailing number without designator in %q", orig)
		}
		return nil
	}

	if err := parse(datePart, map[byte]time.Duration{'D': 24 * time.Hour}); err != nil {
		return 0, err
	}
	if hasT {
		if timePart == "" {
			return 0, fmt.Errorf("empty time component in %q", orig)
		}
		if err := parse(timePart, map[byte]time.Duration{
			'H': time.Hour, 'M': time.Minute, 'S': time.Second,
		}); err != nil {
			return 0, err
		}
	}
	if total == 0 && datePart == "" && !hasT {
		return 0, fmt.Errorf("empty duration %q", orig)
	}
	if neg {
		total = -total
	}
	return total, nil
}
