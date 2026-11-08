package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// appDir is the per-application subdirectory name used under every base path.
const appDir = "vigil"

// Scope determines whether files are resolved against per-user or
// system-wide (all users) locations.
type Scope string

const (
	// ScopeUser resolves paths under the current user's config/cache dirs.
	ScopeUser Scope = "user"
	// ScopeSystem resolves paths under OS-wide machine locations.
	ScopeSystem Scope = "system"
)

// Valid reports whether s is a recognised scope.
func (s Scope) Valid() bool { return s == ScopeUser || s == ScopeSystem }

// ConfigDir returns the directory that holds config.json for the given scope.
//
// User scope follows the XDG convention on every OS (including macOS, where we
// deliberately use ~/.config rather than ~/Library/Application Support so the
// location is predictable and easy to find):
//   - $XDG_CONFIG_HOME/vigil if set
//   - Windows: %AppData%\vigil
//   - otherwise: ~/.config/vigil
func ConfigDir(scope Scope) (string, error) {
	if scope == ScopeSystem {
		return systemDataDir(), nil
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appDir), nil
	}
	if runtime.GOOS == "windows" {
		base, err := os.UserConfigDir() // %AppData%
		if err != nil {
			return "", err
		}
		return filepath.Join(base, appDir), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", appDir), nil
}

// ConfigPath returns the absolute path to config.json for the given scope.
func ConfigPath(scope Scope) (string, error) {
	dir, err := ConfigDir(scope)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// RuntimeDir returns the directory used for mutable runtime artifacts:
// status.json, override.json, the single-instance lock, logs and the
// Graph token cache.
//
// For user scope these live under the XDG state dir so an unprivileged CLI can
// always write the override file that the daemon watches:
//   - $XDG_STATE_HOME/vigil if set
//   - Windows: %LocalAppData%\vigil
//   - otherwise: ~/.local/state/vigil
//
// For system scope they live alongside the system config (writable only by the
// service account / root) — see docs/configuration.md for the multi-user caveat.
func RuntimeDir(scope Scope) (string, error) {
	if scope == ScopeSystem {
		return systemDataDir(), nil
	}
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, appDir), nil
	}
	if runtime.GOOS == "windows" {
		base, err := os.UserCacheDir() // %LocalAppData%
		if err != nil {
			return "", err
		}
		return filepath.Join(base, appDir), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", appDir), nil
}

// TokenPath returns the absolute path to the Graph token cache.
func TokenPath(scope Scope) (string, error) {
	dir, err := RuntimeDir(scope)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "token.json"), nil
}

// systemDataDir returns the OS-wide data directory for the application.
func systemDataDir() string {
	switch runtime.GOOS {
	case "windows":
		if pd := os.Getenv("ProgramData"); pd != "" {
			return filepath.Join(pd, appDir)
		}
		return filepath.Join(`C:\ProgramData`, appDir)
	case "darwin":
		return filepath.Join("/Library/Application Support", appDir)
	default: // linux and other unixes
		return filepath.Join("/etc", appDir)
	}
}
