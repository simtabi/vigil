// Package control implements the file-based control plane shared between the
// daemon and the CLI/TUI: the daemon publishes status.json each tick, clients
// write override.json (watched by the daemon), and a single-instance lock
// prevents two daemons running against the same runtime directory.
package control

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// File names within the runtime directory.
const (
	StatusFile   = "status.json"
	OverrideFile = "override.json"
	LockFile     = "daemon.lock"
	LogFile      = "vigil.log"
)

// Status is the daemon's published state, read by `status` and the TUI.
type Status struct {
	PID           int        `json:"pid"`
	Engine        string     `json:"engine"`
	Activators    []string   `json:"activators"`
	DesiredActive bool       `json:"desired_active"`
	OverrideMode  string     `json:"override_mode"`
	OverrideUntil *time.Time `json:"override_until,omitempty"`
	LastTick      *time.Time `json:"last_tick,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	NextChange    *time.Time `json:"next_change,omitempty"`
	NextActive    bool       `json:"next_active"`
	UpdatedAt     time.Time  `json:"updated_at"`
	StartedAt     time.Time  `json:"started_at"`
}

// StatusPath returns the status.json path within runtimeDir.
func StatusPath(runtimeDir string) string { return filepath.Join(runtimeDir, StatusFile) }

// OverridePath returns the override.json path within runtimeDir.
func OverridePath(runtimeDir string) string { return filepath.Join(runtimeDir, OverrideFile) }

// LockPath returns the lock file path within runtimeDir.
func LockPath(runtimeDir string) string { return filepath.Join(runtimeDir, LockFile) }

// LogPath returns the log file path within runtimeDir.
func LogPath(runtimeDir string) string { return filepath.Join(runtimeDir, LogFile) }

// WriteStatus atomically writes the status to runtimeDir/status.json.
func WriteStatus(runtimeDir string, s Status) error {
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := StatusPath(runtimeDir)
	tmp, err := os.CreateTemp(runtimeDir, ".st-*")
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

// ErrNoStatus indicates the daemon has not published a status yet.
var ErrNoStatus = errors.New("no daemon status found (is the service running?)")

// ReadStatus reads runtimeDir/status.json.
func ReadStatus(runtimeDir string) (Status, error) {
	var s Status
	data, err := os.ReadFile(StatusPath(runtimeDir))
	if errors.Is(err, os.ErrNotExist) {
		return s, ErrNoStatus
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, err
	}
	return s, nil
}

// Lock represents a held single-instance lock.
type Lock struct{ fl *flock.Flock }

// AcquireLock takes an exclusive non-blocking lock in runtimeDir. It returns a
// nil Lock and false if another daemon already holds it.
func AcquireLock(runtimeDir string) (*Lock, bool, error) {
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return nil, false, err
	}
	fl := flock.New(LockPath(runtimeDir))
	ok, err := fl.TryLock()
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	return &Lock{fl: fl}, true, nil
}

// Release releases the lock.
func (l *Lock) Release() error {
	if l == nil || l.fl == nil {
		return nil
	}
	return l.fl.Unlock()
}
