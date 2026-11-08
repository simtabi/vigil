// Package activity contains the pluggable "activator" strategies that keep a
// Teams session showing as Available: synthetic input (per-OS backends) and the
// Microsoft Graph preferred-presence strategy.
package activity

import (
	"context"

	"github.com/simtabi/vigil/internal/config"
)

// Activator is one strategy for keeping the session active.
//
// Tick performs a single activation pulse and is called by the engine on the
// configured interval while the schedule is active. Stop is called when the
// schedule transitions to inactive (and on shutdown) so the activator can
// release any held resources — power assertions, virtual input devices, or a
// sticky Graph presence.
type Activator interface {
	// Name is a short identifier used in status and logs.
	Name() string
	// Tick performs one activation pulse.
	Tick(ctx context.Context) error
	// Stop releases resources and reverts any externally-visible state.
	Stop(ctx context.Context) error
}

// NewInput builds the OS-native synthetic-input activator. It returns an error
// on platforms without a backend or when device/permission setup fails.
func NewInput(cfg config.InputConfig) (Activator, error) {
	return newInputActivator(cfg)
}

// InputSupported reports whether this build has a working synthetic-input
// backend for the current OS.
func InputSupported() bool { return inputSupported }
