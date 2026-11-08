//go:build !windows && !darwin && !linux

package activity

import (
	"errors"

	"github.com/simtabi/vigil/internal/config"
)

const inputSupported = false

func newInputActivator(_ config.InputConfig) (Activator, error) {
	return nil, errors.New("synthetic-input engine is not supported on this OS; use the graph engine")
}

// AccessibilityTrusted is a no-op on unsupported platforms.
func AccessibilityTrusted() bool { return true }
