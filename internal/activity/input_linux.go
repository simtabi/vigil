//go:build linux

package activity

import (
	"context"
	"fmt"
	"sync"

	"github.com/bendahl/uinput"
	"github.com/simtabi/ms-teams-activity/internal/config"
)

const inputSupported = true

// linuxInput drives a /dev/uinput virtual mouse. Because uinput injects real
// kernel input events, it reliably resets the idle timer under both X11 and
// Wayland — unlike synthetic events on macOS. All configured methods map to a
// tiny relative mouse move (the most reliable real event); prevent_sleep is a
// no-op because real input already defers the screensaver.
type linuxInput struct {
	mu    sync.Mutex
	mouse uinput.Mouse
}

func newInputActivator(_ config.InputConfig) (Activator, error) {
	mouse, err := uinput.CreateMouse("/dev/uinput", []byte("mta-virtual-mouse"))
	if err != nil {
		return nil, fmt.Errorf("create uinput virtual mouse (is /dev/uinput present and writable? add your user to a uinput group): %w", err)
	}
	return &linuxInput{mouse: mouse}, nil
}

func (l *linuxInput) Name() string { return "input(linux:uinput)" }

func (l *linuxInput) Tick(_ context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.mouse.MoveRight(1); err != nil {
		return err
	}
	return l.mouse.MoveLeft(1)
}

func (l *linuxInput) Stop(_ context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.mouse != nil {
		err := l.mouse.Close()
		l.mouse = nil
		return err
	}
	return nil
}

// AccessibilityTrusted is a no-op on Linux (always trusted).
func AccessibilityTrusted() bool { return true }
