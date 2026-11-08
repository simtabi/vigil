package selfupdate

import "strings"

// Channel is how the binary was installed, which determines whether in-place
// self-update is appropriate.
type Channel int

const (
	// Standalone — installed via the install script, `vigil self install`, or a
	// downloaded release archive. Safe to self-update.
	Standalone Channel = iota
	// Homebrew — managed by `brew`.
	Homebrew
	// Scoop — managed by Scoop on Windows.
	Scoop
	// SystemPackage — installed via a Linux package manager (deb/rpm).
	SystemPackage
)

// DetectChannel guesses the install channel from the executable path.
func DetectChannel(exePath string) Channel {
	p := strings.ToLower(exePath)
	switch {
	case strings.Contains(p, "/cellar/") || strings.Contains(p, "/homebrew/") || strings.Contains(p, "/.linuxbrew/"):
		return Homebrew
	case strings.Contains(p, `\scoop\`) || strings.Contains(p, "/scoop/"):
		return Scoop
	case strings.HasPrefix(p, "/usr/bin/") || strings.HasPrefix(p, "/bin/"):
		return SystemPackage
	default:
		return Standalone
	}
}

// String returns the channel's identifier (for JSON output).
func (c Channel) String() string {
	switch c {
	case Homebrew:
		return "homebrew"
	case Scoop:
		return "scoop"
	case SystemPackage:
		return "system-package"
	default:
		return "standalone"
	}
}

// SelfUpdatable reports whether `vigil upgrade` should replace this binary.
func (c Channel) SelfUpdatable() bool { return c == Standalone }

// Advice returns guidance for updating via the detected package manager.
func (c Channel) Advice() string {
	switch c {
	case Homebrew:
		return "this binary is managed by Homebrew — update with `brew upgrade vigil`"
	case Scoop:
		return "this binary is managed by Scoop — update with `scoop update vigil`"
	case SystemPackage:
		return "this binary is managed by your system package manager — update with `apt upgrade` / `dnf upgrade`"
	default:
		return ""
	}
}
