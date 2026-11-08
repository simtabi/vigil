package tz

import (
	"strings"
	"testing"
	"time"
)

func TestZonesArePrepopulatedAndValid(t *testing.T) {
	if len(Zones) < 100 {
		t.Fatalf("expected a large prepopulated zone list, got %d", len(Zones))
	}
	// Spot-check a few load via the standard library.
	for _, z := range []string{"UTC", "America/New_York", "Europe/London", "Asia/Tokyo"} {
		if IndexOf(z) < 0 {
			t.Errorf("expected %q in the zone list", z)
		}
		if _, err := time.LoadLocation(z); err != nil {
			t.Errorf("zone %q should load: %v", z, err)
		}
	}
}

func TestFilter(t *testing.T) {
	// Space/underscore-insensitive, multi-term.
	got := Filter("new york")
	if len(got) == 0 || !contains(got, "America/New_York") {
		t.Fatalf("'new york' should match America/New_York, got %v", got)
	}
	// Case-insensitive single term.
	if !contains(Filter("LONDON"), "Europe/London") {
		t.Error("'LONDON' should match Europe/London")
	}
	// Empty query returns everything.
	if len(Filter("")) != len(Zones) {
		t.Error("empty query should return all zones")
	}
	// No match.
	if len(Filter("zzznotazone")) != 0 {
		t.Error("nonsense query should return nothing")
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want || strings.EqualFold(s, want) {
			return true
		}
	}
	return false
}
