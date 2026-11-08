package config

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// field describes a single settable/gettable config key.
type field struct {
	get func(c Config) string
	set func(c *Config, v string) error
}

// registry maps dotted keys to typed accessors. Schedule windows are managed
// separately (TUI editor / `schedule` commands), not via this flat registry.
var registry = map[string]field{
	"engine": {
		func(c Config) string { return string(c.Engine) },
		func(c *Config, v string) error { c.Engine = Engine(v); return nil },
	},
	"timezone": {
		func(c Config) string { return c.Timezone },
		func(c *Config, v string) error { c.Timezone = v; return nil },
	},
	"schedule.enabled": {
		func(c Config) string { return strconv.FormatBool(c.Schedule.Enabled) },
		func(c *Config, v string) error { return setBool(&c.Schedule.Enabled, v) },
	},
	"schedule.always": {
		func(c Config) string { return strconv.FormatBool(c.Schedule.Always) },
		func(c *Config, v string) error { return setBool(&c.Schedule.Always, v) },
	},
	"input.interval_seconds": {
		func(c Config) string { return strconv.Itoa(c.Input.IntervalSeconds) },
		func(c *Config, v string) error { return setInt(&c.Input.IntervalSeconds, v) },
	},
	"input.jitter_seconds": {
		func(c Config) string { return strconv.Itoa(c.Input.JitterSeconds) },
		func(c *Config, v string) error { return setInt(&c.Input.JitterSeconds, v) },
	},
	"input.method": {
		func(c Config) string { return string(c.Input.Method) },
		func(c *Config, v string) error { c.Input.Method = InputMethod(v); return nil },
	},
	"input.prevent_sleep": {
		func(c Config) string { return strconv.FormatBool(c.Input.PreventSleep) },
		func(c *Config, v string) error { return setBool(&c.Input.PreventSleep, v) },
	},
	"input.move_pixels": {
		func(c Config) string { return strconv.Itoa(c.Input.MovePixels) },
		func(c *Config, v string) error { return setInt(&c.Input.MovePixels, v) },
	},
	"graph.tenant_id": {
		func(c Config) string { return c.Graph.TenantID },
		func(c *Config, v string) error { c.Graph.TenantID = v; return nil },
	},
	"graph.client_id": {
		func(c Config) string { return c.Graph.ClientID },
		func(c *Config, v string) error { c.Graph.ClientID = v; return nil },
	},
	"graph.availability": {
		func(c Config) string { return c.Graph.Availability },
		func(c *Config, v string) error { c.Graph.Availability = v; return nil },
	},
	"graph.activity": {
		func(c Config) string { return c.Graph.Activity },
		func(c *Config, v string) error { c.Graph.Activity = v; return nil },
	},
	"graph.expiration": {
		func(c Config) string { return c.Graph.Expiration },
		func(c *Config, v string) error { c.Graph.Expiration = v; return nil },
	},
	"graph.refresh_minutes": {
		func(c Config) string { return strconv.Itoa(c.Graph.RefreshMinutes) },
		func(c *Config, v string) error { return setInt(&c.Graph.RefreshMinutes, v) },
	},
	"control.port": {
		func(c Config) string { return strconv.Itoa(c.Control.Port) },
		func(c *Config, v string) error { return setInt(&c.Control.Port, v) },
	},
	"log.level": {
		func(c Config) string { return c.Log.Level },
		func(c *Config, v string) error { c.Log.Level = v; return nil },
	},
	"log.max_size_mb": {
		func(c Config) string { return strconv.Itoa(c.Log.MaxSizeMB) },
		func(c *Config, v string) error { return setInt(&c.Log.MaxSizeMB, v) },
	},
	"log.max_backups": {
		func(c Config) string { return strconv.Itoa(c.Log.MaxBackups) },
		func(c *Config, v string) error { return setInt(&c.Log.MaxBackups, v) },
	},
}

// SettableKeys returns the sorted list of supported dotted keys.
func SettableKeys() []string {
	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GetField returns the string value of a dotted key.
func (c Config) GetField(key string) (string, error) {
	f, ok := registry[key]
	if !ok {
		return "", unknownKey(key)
	}
	return f.get(c), nil
}

// SetField sets a dotted key from a string and re-validates the whole config.
// On validation failure the config is left unchanged.
func (c *Config) SetField(key, value string) error {
	f, ok := registry[key]
	if !ok {
		return unknownKey(key)
	}
	candidate := *c
	if err := f.set(&candidate, value); err != nil {
		return fmt.Errorf("set %s: %w", key, err)
	}
	if err := candidate.Validate(); err != nil {
		return fmt.Errorf("set %s=%q would make the config invalid: %w", key, value, err)
	}
	*c = candidate
	return nil
}

func unknownKey(key string) error {
	return fmt.Errorf("unknown config key %q (valid keys: %s)", key, strings.Join(SettableKeys(), ", "))
}

func setBool(dst *bool, v string) error {
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return fmt.Errorf("expected a boolean (true/false), got %q", v)
	}
	*dst = b
	return nil
}

func setInt(dst *int, v string) error {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return fmt.Errorf("expected an integer, got %q", v)
	}
	*dst = n
	return nil
}
