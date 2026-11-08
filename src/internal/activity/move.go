package activity

import "math/rand"

// naturalDelta returns a small, non-zero pixel offset (magnitude 1..maxPx) in a
// random direction. Using a varied offset each tick produces less mechanical,
// more natural cursor movement than a fixed step. The activator moves by this
// delta and immediately back, so the cursor does not drift.
func naturalDelta(maxPx int) int {
	if maxPx < 1 {
		maxPx = 1
	}
	d := 1 + rand.Intn(maxPx)
	if rand.Intn(2) == 0 {
		return -d
	}
	return d
}

// naturalVertical reports whether this tick should nudge vertically instead of
// horizontally, so movement isn't always along one axis.
func naturalVertical() bool { return rand.Intn(4) == 0 }
