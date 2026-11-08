package cli

import (
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/simtabi/vigil/internal/cli/ui"
	"github.com/simtabi/vigil/internal/config"
	"github.com/simtabi/vigil/internal/service"
)

// binName is the platform binary filename.
func binName() string {
	if runtime.GOOS == "windows" {
		return "vigil.exe"
	}
	return "vigil"
}

// binInstallDir returns the directory `self install` copies the binary into.
func binInstallDir(s config.Scope) (string, error) {
	if runtime.GOOS == "windows" {
		if s == config.ScopeSystem {
			return filepath.Join(os.Getenv("ProgramFiles"), "vigil"), nil
		}
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			base = home
		}
		return filepath.Join(base, "Programs", "vigil"), nil
	}
	if s == config.ScopeSystem {
		return "/usr/local/bin", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "bin"), nil
}

// copyFile copies src to dst with the given permission bits.
func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	// Clean up the partial temp file on any failure (no-op after a successful rename).
	defer func() { _ = os.Remove(tmp) }()
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

// serviceParamsBestEffort builds service params if a config exists.
func serviceParamsBestEffort() (service.Params, bool) {
	cfg, err := loadConfig()
	if err != nil {
		return service.Params{}, false
	}
	cfgPath, err := configPath()
	if err != nil {
		return service.Params{}, false
	}
	return service.Params{Scope: scope(), ConfigPath: cfgPath, UsesInput: cfg.UsesInput()}, true
}

// tccReminderIfNeeded prints the macOS Accessibility re-grant note after a
// binary change, when the input engine is in use.
func tccReminderIfNeeded() {
	if runtime.GOOS != "darwin" {
		return
	}
	if cfg, err := loadConfig(); err == nil && cfg.UsesInput() {
		ui.Warn("the binary changed — macOS may require re-granting Accessibility (System Settings → Privacy & Security → Accessibility). Run `vigil doctor` to confirm.")
	}
}
