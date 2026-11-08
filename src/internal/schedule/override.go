package schedule

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// OverrideMode is a manual override of the schedule.
type OverrideMode string

const (
	// OverrideNone means "follow the schedule".
	OverrideNone OverrideMode = ""
	// OverrideOn forces the daemon active.
	OverrideOn OverrideMode = "on"
	// OverrideOff forces the daemon inactive.
	OverrideOff OverrideMode = "off"
)

// Override is the persisted manual override. A nil Until is indefinite.
type Override struct {
	Mode  OverrideMode `json:"mode"`
	Until *time.Time   `json:"until,omitempty"`
	SetAt time.Time    `json:"set_at"`
}

// EffectiveMode returns the override mode in force at time t, treating an
// expired (Until <= t) override as OverrideNone.
func (o Override) EffectiveMode(t time.Time) OverrideMode {
	if o.Mode == OverrideNone {
		return OverrideNone
	}
	if o.Until != nil && !t.Before(*o.Until) {
		return OverrideNone
	}
	return o.Mode
}

// Active reports whether the override is currently in force at t.
func (o Override) Active(t time.Time) bool { return o.EffectiveMode(t) != OverrideNone }

// LoadOverride reads override.json. A missing file yields a zero (None) override
// and no error.
func LoadOverride(path string) (Override, error) {
	var o Override
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Override{}, nil
	}
	if err != nil {
		return o, err
	}
	if err := json.Unmarshal(data, &o); err != nil {
		return Override{}, err
	}
	return o, nil
}

// SaveOverride atomically writes the override to path.
func SaveOverride(path string, o Override) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".ovr-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
