//go:build windows

package activity

import (
	"context"
	"fmt"
	"sync"
	"unsafe"

	"github.com/simtabi/vigil/internal/config"
	"golang.org/x/sys/windows"
)

const inputSupported = true

const (
	inputMouse    = 0
	inputKeyboard = 1

	mouseeventfMove = 0x0001

	keyeventfKeyup = 0x0002
	vkF15          = 0x7E

	esContinuous      = 0x80000000
	esSystemRequired  = 0x00000001
	esDisplayRequired = 0x00000002
)

var (
	user32                     = windows.NewLazySystemDLL("user32.dll")
	kernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	procSendInput              = user32.NewProc("SendInput")
	procSetThreadExecutionStat = kernel32.NewProc("SetThreadExecutionState")
)

// input mirrors the Win32 INPUT union sized for its largest member (MOUSEINPUT)
// on amd64 (40 bytes). The keyboard fields are overlaid via a separate type of
// identical size.
type mouseInputUnion struct {
	typ uint32
	_   [4]byte
	mi  rawMouseInput
}

type rawMouseInput struct {
	dx          int32
	dy          int32
	mouseData   uint32
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

type keybdInputUnion struct {
	typ uint32
	_   [4]byte
	ki  rawKeybdInput
	_   [8]byte
}

type rawKeybdInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

type windowsInput struct {
	method       config.InputMethod
	preventSleep bool
	movePx       int

	mu   sync.Mutex
	held bool
}

func newInputActivator(cfg config.InputConfig) (Activator, error) {
	return &windowsInput{method: cfg.Method, preventSleep: cfg.PreventSleep, movePx: cfg.MovePixels}, nil
}

func (w *windowsInput) Name() string { return "input(windows:sendinput)" }

func (w *windowsInput) Tick(_ context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Assert once; the engine goroutine is pinned to its OS thread (see
	// engine.Run) so this thread-affine state is set and later cleared on the
	// same thread.
	if w.preventSleep && !w.held {
		_, _, _ = procSetThreadExecutionStat.Call(uintptr(esContinuous | esSystemRequired | esDisplayRequired))
		w.held = true
	}

	switch w.method {
	case config.MethodKey:
		return sendKey(vkF15)
	case config.MethodZen:
		return sendMouseMove(0, 0)
	default: // MethodMouse — a real, varied small move that returns to origin.
		d := int32(naturalDelta(w.movePx))
		dx, dy := d, int32(0)
		if naturalVertical() {
			dx, dy = 0, d
		}
		if err := sendMouseMove(dx, dy); err != nil {
			return err
		}
		return sendMouseMove(-dx, -dy)
	}
}

func (w *windowsInput) Stop(_ context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.held {
		_, _, _ = procSetThreadExecutionStat.Call(uintptr(esContinuous))
		w.held = false
	}
	return nil
}

func sendMouseMove(dx, dy int32) error {
	in := mouseInputUnion{typ: inputMouse}
	in.mi = rawMouseInput{dx: dx, dy: dy, dwFlags: mouseeventfMove}
	return send(unsafe.Pointer(&in))
}

func sendKey(vk uint16) error {
	down := keybdInputUnion{typ: inputKeyboard}
	down.ki = rawKeybdInput{wVk: vk}
	if err := send(unsafe.Pointer(&down)); err != nil {
		return err
	}
	up := keybdInputUnion{typ: inputKeyboard}
	up.ki = rawKeybdInput{wVk: vk, dwFlags: keyeventfKeyup}
	return send(unsafe.Pointer(&up))
}

func send(p unsafe.Pointer) error {
	const size = unsafe.Sizeof(mouseInputUnion{})
	n, _, err := procSendInput.Call(1, uintptr(p), size)
	if n != 1 {
		return fmt.Errorf("SendInput injected %d of 1 events: %w", n, err)
	}
	return nil
}

// AccessibilityTrusted is a no-op on Windows (always trusted).
func AccessibilityTrusted() bool { return true }
