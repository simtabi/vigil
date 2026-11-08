package engine

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/simtabi/vigil/internal/activity"
	"github.com/simtabi/vigil/internal/config"
	"github.com/simtabi/vigil/internal/control"
	"github.com/simtabi/vigil/internal/schedule"
)

type fakeActivator struct {
	mu           sync.Mutex
	ticks, stops int
}

func (f *fakeActivator) Name() string { return "fake" }
func (f *fakeActivator) Tick(context.Context) error {
	f.mu.Lock()
	f.ticks++
	f.mu.Unlock()
	return nil
}
func (f *fakeActivator) Stop(context.Context) error {
	f.mu.Lock()
	f.stops++
	f.mu.Unlock()
	return nil
}

func newTestEngine(t *testing.T, cfg config.Config) *Engine {
	t.Helper()
	rt := t.TempDir()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	e := New(config.ScopeUser, filepath.Join(rt, "config.json"), rt, filepath.Join(rt, "token.json"), false, log)
	e.cfg = cfg
	return e
}

func TestEngineDrivesWhenActive(t *testing.T) {
	cfg := config.Default()
	cfg.Schedule.Always = true
	e := newTestEngine(t, cfg)
	fake := &fakeActivator{}
	e.activators = []activity.Activator{fake}

	e.cycle(context.Background())

	if fake.ticks != 1 {
		t.Fatalf("expected 1 tick while active, got %d", fake.ticks)
	}
	st, err := control.ReadStatus(e.runtimeDir)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	if !st.DesiredActive {
		t.Fatal("status should report DesiredActive=true")
	}
}

func TestEngineStopsOnTransitionToInactive(t *testing.T) {
	cfg := config.Default()
	cfg.Schedule.Enabled = false // never active by schedule
	e := newTestEngine(t, cfg)
	fake := &fakeActivator{}
	e.activators = []activity.Activator{fake}
	e.activeNow = true // pretend we were active last cycle

	e.cycle(context.Background())

	if fake.stops != 1 {
		t.Fatalf("expected 1 stop on deactivation, got %d", fake.stops)
	}
	if fake.ticks != 0 {
		t.Fatalf("expected 0 ticks while inactive, got %d", fake.ticks)
	}
}

func TestEngineOverrideOffSuppressesActivation(t *testing.T) {
	cfg := config.Default()
	cfg.Schedule.Always = true
	e := newTestEngine(t, cfg)
	fake := &fakeActivator{}
	e.activators = []activity.Activator{fake}

	if err := schedule.SaveOverride(control.OverridePath(e.runtimeDir),
		schedule.Override{Mode: schedule.OverrideOff, SetAt: time.Now()}); err != nil {
		t.Fatalf("save override: %v", err)
	}

	e.cycle(context.Background())

	if fake.ticks != 0 {
		t.Fatalf("override off must suppress ticks, got %d", fake.ticks)
	}
}

func TestTickIntervalRespectsJitterBounds(t *testing.T) {
	cfg := config.Default()
	cfg.Input.IntervalSeconds = 60
	cfg.Input.JitterSeconds = 10
	e := newTestEngine(t, cfg)
	lo, hi := 50*time.Second, 70*time.Second
	for i := 0; i < 100; i++ {
		d := e.tickInterval()
		if d < lo || d > hi {
			t.Fatalf("interval %v out of [%v,%v]", d, lo, hi)
		}
	}
}
