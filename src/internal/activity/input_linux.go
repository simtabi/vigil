//go:build linux

package activity

import (
	"context"
	"fmt"
	"sync"

	"github.com/bendahl/uinput"
	"github.com/simtabi/vigil/internal/config"
)

const inputSupported = true

// linuxInput drives a /dev/uinput virtual mouse. Because uinput injects real
// kernel input events, it reliably resets the idle timer under both X11 and
// Wayland — unlike synthetic events on macOS. All configured methods map to a
// tiny relative mouse move (the most reliable real event); prevent_sleep is a
// no-op because real input already defers the screensaver.
type linuxInput struct {
	mu     sync.Mutex
	mouse  uinput.Mouse
	movePx int
}

func newInputActivator(cfg config.InputConfig) (Activator, error) {
	mouse, err := uinput.CreateMouse("/dev/uinput", []byte("vigil-virtual-mouse"))
	if err != nil {
		return nil, fmt.Errorf("create uinput virtual mouse (is /dev/uinput present and writable? add your user to a uinput group): %w", err)
	}
	return &linuxInput{mouse: mouse, movePx: cfg.MovePixels}, nil
}

func (l *linuxInput) Name() string { return "input(linux:uinput)" }

func (l *linuxInput) Tick(_ context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := int32(abs(naturalDelta(l.movePx))) // varied magnitude 1..move_pixels
	if naturalVertical() {
		if err := l.mouse.MoveUp(n); err != nil {
			return err
		}
		return l.mouse.MoveDown(n)
	}
	if err := l.mouse.MoveRight(n); err != nil {
		return err
	}
	return l.mouse.MoveLeft(n)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
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
