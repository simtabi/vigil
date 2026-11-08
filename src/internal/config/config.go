// Package config defines the on-disk JSON configuration for vigil,
// along with defaults, validation and atomic load/save helpers.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// CurrentVersion is the config schema version written by this build. It allows
// future migrations to detect and upgrade older files.
const CurrentVersion = 1

// Engine selects which activation strategy (or strategies) the daemon runs.
type Engine string

const (
	// EngineInput keeps the session active with synthetic input (default).
	EngineInput Engine = "input"
	// EngineGraph sets a sticky presence via Microsoft Graph.
	EngineGraph Engine = "graph"
	// EngineBoth runs the input and graph strategies together.
	EngineBoth Engine = "both"
)

// InputMethod selects how synthetic input is generated.
type InputMethod string

const (
	// MethodMouse performs a small real relative mouse move (default, reliable).
	MethodMouse InputMethod = "mouse"
	// MethodKey presses a benign key (F15) that has no side effects.
	MethodKey InputMethod = "key"
	// MethodZen requests an invisible/zero-delta nudge. May be ignored by apps
	// that implement their own idle detection — opt-in only.
	MethodZen InputMethod = "zen"
)

// Config is the root configuration object persisted as config.json.
type Config struct {
	Version  int            `json:"version"`
	Engine   Engine         `json:"engine"`
	Timezone string         `json:"timezone"`
	Schedule ScheduleConfig `json:"schedule"`
	Input    InputConfig    `json:"input"`
	Graph    GraphConfig    `json:"graph"`
	Control  ControlConfig  `json:"control"`
	Log      LogConfig      `json:"log"`
}

// ScheduleConfig describes when the daemon should keep Teams active.
type ScheduleConfig struct {
	// Enabled turns schedule evaluation on. When false the daemon stays idle
	// unless a manual override forces it active.
	Enabled bool `json:"enabled"`
	// Always keeps the daemon active whenever it runs, ignoring Windows.
	Always bool `json:"always"`
	// Windows are the active periods, evaluated in Config.Timezone.
	Windows []Window `json:"windows"`
}

// Window is a recurring weekly active period. Start/End are "HH:MM" 24h local
// times; an End that is not after Start denotes an overnight window.
type Window struct {
	Days  []string `json:"days"`  // e.g. ["Mon","Tue",...]; "Mon".."Sun"
	Start string   `json:"start"` // "HH:MM"
	End   string   `json:"end"`   // "HH:MM"
}

// InputConfig tunes the synthetic-input engine.
type InputConfig struct {
	IntervalSeconds int         `json:"interval_seconds"`
	JitterSeconds   int         `json:"jitter_seconds"`
	Method          InputMethod `json:"method"`
	PreventSleep    bool        `json:"prevent_sleep"`
	// MovePixels is the maximum magnitude (px) of the mouse nudge; each tick
	// uses a random 1..MovePixels offset and returns to origin.
	MovePixels int `json:"move_pixels"`
}

// GraphConfig holds Microsoft Graph engine settings.
type GraphConfig struct {
	TenantID       string `json:"tenant_id"`
	ClientID       string `json:"client_id"`
	Availability   string `json:"availability"`
	Activity       string `json:"activity"`
	Expiration     string `json:"expiration"` // ISO-8601 duration, e.g. "PT8H"
	RefreshMinutes int    `json:"refresh_minutes"`
}

// ControlConfig configures the control plane. Port 0 means file-based control.
type ControlConfig struct {
	Port int `json:"port"`
}

// LogConfig configures the rotating log file.
type LogConfig struct {
	Level      string `json:"level"`
	MaxSizeMB  int    `json:"max_size_mb"`
	MaxBackups int    `json:"max_backups"`
}

var validDays = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

// Default returns a Config populated with sensible defaults (Mon–Fri 08:00–17:00,
// synthetic-input engine).
func Default() Config {
	return Config{
		Version:  CurrentVersion,
		Engine:   EngineInput,
		Timezone: "Local",
		Schedule: ScheduleConfig{
			Enabled: true,
			Always:  false,
			Windows: []Window{
				{Days: []string{"Mon", "Tue", "Wed", "Thu", "Fri"}, Start: "08:00", End: "17:00"},
			},
		},
		Input: InputConfig{
			IntervalSeconds: 60,
			JitterSeconds:   25,
			Method:          MethodMouse,
			PreventSleep:    true,
			MovePixels:      3,
		},
		Graph: GraphConfig{
			TenantID:       "common",
			ClientID:       "",
			Availability:   "Available",
			Activity:       "Available",
			Expiration:     "PT8H",
			RefreshMinutes: 60,
		},
		Control: ControlConfig{Port: 0},
		Log:     LogConfig{Level: "info", MaxSizeMB: 5, MaxBackups: 3},
	}
}

// Load reads and validates the config at path. A missing file is reported via
// os.IsNotExist on the returned error so callers can offer `config init`.
func Load(path string) (Config, error) {
	var c Config
	data, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return c, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return c, nil
}

// Save atomically writes c to path (temp file + rename), creating parent dirs.
func (c Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicWrite(path, data, 0o644)
}

// atomicWrite writes data to a sibling temp file then renames it over path.
func atomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// Validate checks the config for internal consistency and returns the first
// problem found.
func (c Config) Validate() error {
	if c.Version <= 0 || c.Version > CurrentVersion {
		return fmt.Errorf("unsupported version %d (this build supports up to %d)", c.Version, CurrentVersion)
	}
	switch c.Engine {
	case EngineInput, EngineGraph, EngineBoth:
	default:
		return fmt.Errorf("engine: must be one of input|graph|both, got %q", c.Engine)
	}
	if _, err := c.Location(); err != nil {
		return fmt.Errorf("timezone %q: %w", c.Timezone, err)
	}
	if err := c.Schedule.validate(); err != nil {
		return err
	}
	if c.usesInput() {
		if err := c.Input.validate(); err != nil {
			return err
		}
	}
	if c.usesGraph() {
		if err := c.Graph.validate(); err != nil {
			return err
		}
	}
	if c.Control.Port < 0 || c.Control.Port > 65535 {
		return fmt.Errorf("control.port: out of range: %d", c.Control.Port)
	}
	return nil
}

func (c Config) usesInput() bool { return c.Engine == EngineInput || c.Engine == EngineBoth }
func (c Config) usesGraph() bool { return c.Engine == EngineGraph || c.Engine == EngineBoth }

// UsesInput reports whether the configured engine runs the input strategy.
func (c Config) UsesInput() bool { return c.usesInput() }

// UsesGraph reports whether the configured engine runs the Graph strategy.
func (c Config) UsesGraph() bool { return c.usesGraph() }

// Location resolves the configured timezone. "Local" and "" map to time.Local.
func (c Config) Location() (*time.Location, error) {
	if c.Timezone == "" || c.Timezone == "Local" {
		return time.Local, nil
	}
	return time.LoadLocation(c.Timezone)
}

func (s ScheduleConfig) validate() error {
	for i, w := range s.Windows {
		if _, err := ParseClock(w.Start); err != nil {
			return fmt.Errorf("schedule.windows[%d].start: %w", i, err)
		}
		if _, err := ParseClock(w.End); err != nil {
			return fmt.Errorf("schedule.windows[%d].end: %w", i, err)
		}
		if len(w.Days) == 0 {
			return fmt.Errorf("schedule.windows[%d]: at least one day required", i)
		}
		for _, d := range w.Days {
			if !slices.Contains(validDays, normalizeDay(d)) {
				return fmt.Errorf("schedule.windows[%d]: invalid day %q (use Mon..Sun)", i, d)
			}
		}
	}
	return nil
}

func (in InputConfig) validate() error {
	if in.IntervalSeconds < 5 || in.IntervalSeconds >= 300 {
		return fmt.Errorf("input.interval_seconds: must be in [5,300), got %d (Teams idles at ~5 min)", in.IntervalSeconds)
	}
	if in.JitterSeconds < 0 || in.JitterSeconds >= in.IntervalSeconds {
		return fmt.Errorf("input.jitter_seconds: must be in [0, interval), got %d", in.JitterSeconds)
	}
	switch in.Method {
	case MethodMouse, MethodKey, MethodZen:
	default:
		return fmt.Errorf("input.method: must be one of mouse|key|zen, got %q", in.Method)
	}
	if in.MovePixels < 1 || in.MovePixels > 50 {
		return fmt.Errorf("input.move_pixels: must be in [1,50], got %d", in.MovePixels)
	}
	return nil
}

// Supported Graph presence values.
var (
	validAvailability = []string{"Available", "Busy", "DoNotDisturb", "BeRightBack", "Away", "Offline"}
	validActivity     = []string{"Available", "Busy", "DoNotDisturb", "BeRightBack", "Away", "OffWork"}
)

func (g GraphConfig) validate() error {
	if strings.TrimSpace(g.ClientID) == "" {
		return fmt.Errorf("graph.client_id: required when the graph engine is enabled (register an Entra public-client app)")
	}
	if strings.TrimSpace(g.TenantID) == "" {
		return fmt.Errorf("graph.tenant_id: required (use \"common\", \"organizations\" or a tenant GUID)")
	}
	if g.RefreshMinutes < 1 {
		return fmt.Errorf("graph.refresh_minutes: must be >= 1, got %d", g.RefreshMinutes)
	}
	if _, err := ParseISODuration(g.Expiration); err != nil {
		return fmt.Errorf("graph.expiration: %w", err)
	}
	if !slices.Contains(validAvailability, g.Availability) {
		return fmt.Errorf("graph.availability: must be one of %s", strings.Join(validAvailability, ", "))
	}
	if !slices.Contains(validActivity, g.Activity) {
		return fmt.Errorf("graph.activity: must be one of %s", strings.Join(validActivity, ", "))
	}
	return nil
}

// normalizeDay title-cases a 3-letter day abbreviation (e.g. "mon" -> "Mon").
func normalizeDay(d string) string {
	d = strings.TrimSpace(d)
	if len(d) < 3 {
		return d
	}
	d = d[:3]
	return strings.ToUpper(d[:1]) + strings.ToLower(d[1:])
}
