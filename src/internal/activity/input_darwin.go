//go:build darwin

package activity

/*
#cgo LDFLAGS: -framework CoreGraphics -framework IOKit -framework CoreFoundation -framework ApplicationServices
#include <CoreGraphics/CoreGraphics.h>
#include <IOKit/pwr_mgt/IOPMLib.h>
#include <ApplicationServices/ApplicationServices.h>

// nudge posts a synthetic mouse-moved event by (dx,dy), then immediately back to
// the original position so the cursor does not drift. (0,0) posts an in-place
// move ("zen").
static void nudge(int dx, int dy) {
    CGEventRef probe = CGEventCreate(NULL);
    CGPoint p = CGEventGetLocation(probe);
    CFRelease(probe);

    if (dx != 0 || dy != 0) {
        CGPoint moved = CGPointMake(p.x + dx, p.y + dy);
        CGEventRef e1 = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, moved, kCGMouseButtonLeft);
        CGEventPost(kCGHIDEventTap, e1);
        CFRelease(e1);
    }
    CGEventRef e2 = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, p, kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, e2);
    CFRelease(e2);
}

// pressF15 taps the F15 key (virtual keycode 0x71), which has no default action.
static void pressF15(void) {
    CGEventRef down = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)0x71, true);
    CGEventPost(kCGHIDEventTap, down);
    CFRelease(down);
    CGEventRef up = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)0x71, false);
    CGEventPost(kCGHIDEventTap, up);
    CFRelease(up);
}

// preventDisplaySleep takes a power assertion deferring user-idle display sleep
// (which also defers the screensaver/lock on most configs). Returns 0 on failure.
static IOPMAssertionID preventDisplaySleep(void) {
    IOPMAssertionID id = 0;
    CFStringRef reason = CFSTR("vigil keeping session active");
    IOReturn r = IOPMAssertionCreateWithName(
        kIOPMAssertionTypePreventUserIdleDisplaySleep,
        kIOPMAssertionLevelOn, reason, &id);
    if (r != kIOReturnSuccess) {
        return 0;
    }
    return id;
}

static void releaseAssertion(IOPMAssertionID id) {
    if (id != 0) {
        IOPMAssertionRelease(id);
    }
}

static int axTrusted(void) {
    return AXIsProcessTrusted() ? 1 : 0;
}
*/
import "C"

import (
	"context"
	"sync"

	"github.com/simtabi/vigil/internal/config"
)

const inputSupported = true

type darwinInput struct {
	method       config.InputMethod
	preventSleep bool
	movePx       int

	mu          sync.Mutex
	assertionID C.IOPMAssertionID
	held        bool
}

func newInputActivator(cfg config.InputConfig) (Activator, error) {
	return &darwinInput{method: cfg.Method, preventSleep: cfg.PreventSleep, movePx: cfg.MovePixels}, nil
}

func (d *darwinInput) Name() string { return "input(macos:cgevent)" }

func (d *darwinInput) Tick(_ context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.preventSleep && !d.held {
		if id := C.preventDisplaySleep(); id != 0 {
			d.assertionID = id
			d.held = true
		}
	}

	switch d.method {
	case config.MethodKey:
		C.pressF15()
	case config.MethodZen:
		C.nudge(0, 0)
	default: // MethodMouse — varied small offset, random axis, for natural movement
		off := C.int(naturalDelta(d.movePx))
		if naturalVertical() {
			C.nudge(0, off)
		} else {
			C.nudge(off, 0)
		}
	}
	return nil
}

func (d *darwinInput) Stop(_ context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.held {
		C.releaseAssertion(d.assertionID)
		d.held = false
		d.assertionID = 0
	}
	return nil
}

// AccessibilityTrusted reports whether this process is trusted for macOS
// Accessibility, which is required for CGEventPost to take effect.
func AccessibilityTrusted() bool { return C.axTrusted() == 1 }
