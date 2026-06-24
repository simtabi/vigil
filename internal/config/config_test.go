package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultIsValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	want := Default()
	want.Timezone = "America/New_York"
	if err := want.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Timezone != want.Timezone || got.Engine != want.Engine {
		t.Fatalf("round-trip mismatch: got %+v want %+v", got, want)
	}
}

func TestValidate_Errors(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
	}{
		{"bad engine", func(c *Config) { c.Engine = "nope" }},
		{"bad timezone", func(c *Config) { c.Timezone = "Mars/Phobos" }},
		{"interval too high", func(c *Config) { c.Input.IntervalSeconds = 600 }},
		{"jitter >= interval", func(c *Config) { c.Input.JitterSeconds = 60; c.Input.IntervalSeconds = 60 }},
		{"bad method", func(c *Config) { c.Input.Method = "wiggle" }},
		{"graph without client", func(c *Config) { c.Engine = EngineGraph; c.Graph.ClientID = "" }},
		{"bad version", func(c *Config) { c.Version = 99 }},
		{"bad day", func(c *Config) { c.Schedule.Windows[0].Days = []string{"Funday"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := Default()
			tc.mutate(&c)
			if err := c.Validate(); err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestParseISODuration(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"PT8H", 8 * time.Hour, true},
		{"PT30M", 30 * time.Minute, true},
		{"P1DT2H", 26 * time.Hour, true},
		{"PT1H30M", 90 * time.Minute, true},
		{"P1W", 0, false}, // weeks unsupported
		{"8H", 0, false},  // missing P
		{"", 0, false},
	}
	for _, tc := range cases {
		got, err := ParseISODuration(tc.in)
		if tc.ok && (err != nil || got != tc.want) {
			t.Errorf("ParseISODuration(%q) = %v, %v; want %v", tc.in, got, err, tc.want)
		}
		if !tc.ok && err == nil {
			t.Errorf("ParseISODuration(%q) expected error", tc.in)
		}
	}
}

func TestParseClock(t *testing.T) {
	if c, err := ParseClock("08:30"); err != nil || c.Minutes() != 510 {
		t.Fatalf("ParseClock(08:30) = %v, %v", c, err)
	}
	if _, err := ParseClock("25:00"); err == nil {
		t.Fatal("expected error for 25:00")
	}
}
