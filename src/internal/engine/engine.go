// Package engine is the daemon orchestrator: it evaluates the schedule+override
// on a cadence, drives the configured activators, hot-reloads config/override
// via fsnotify, publishes status, and reverts cleanly on shutdown.
package engine

import (
	"context"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/simtabi/vigil/internal/activity"
	"github.com/simtabi/vigil/internal/config"
	"github.com/simtabi/vigil/internal/control"
	"github.com/simtabi/vigil/internal/graph"
	"github.com/simtabi/vigil/internal/schedule"
)

// idlePoll is the wake cadence while inactive, so schedule transitions and
// override changes are picked up promptly.
const idlePoll = 30 * time.Second

// Engine runs the daemon loop for one configuration.
type Engine struct {
	scope      config.Scope
	configPath string
	runtimeDir string
	tokenPath  string
	log        *slog.Logger

	cfg        config.Config
	activators []activity.Activator
	activeNow  bool
	lastTick   *time.Time
	lastErr    string
	startedAt  time.Time
	dryRun     bool
}

// New constructs an Engine. configPath, runtimeDir and tokenPath are resolved by
// the caller from the chosen scope. When dryRun is true the engine logs intended
// actions instead of injecting input or calling Graph.
func New(scope config.Scope, configPath, runtimeDir, tokenPath string, dryRun bool, log *slog.Logger) *Engine {
	return &Engine{
		scope:      scope,
		configPath: configPath,
		runtimeDir: runtimeDir,
		tokenPath:  tokenPath,
		dryRun:     dryRun,
		log:        log,
		startedAt:  time.Now(),
	}
}

// Run blocks until ctx is cancelled, then reverts all activators.
func (e *Engine) Run(ctx context.Context) error {
	// Pin this goroutine to its OS thread for the daemon's lifetime. All activator
	// Tick/Stop calls happen here, and Windows' SetThreadExecutionState (sleep
	// prevention) is thread-affine — without pinning, the goroutine could migrate
	// threads between the "hold" and "release" calls, leaking the assertion.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cfg, err := config.Load(e.configPath)
	if err != nil {
		return err
	}
	e.cfg = cfg
	if err := e.rebuild(ctx); err != nil {
		// Build errors (e.g. uinput perms, no client_id) are fatal at startup so
		// the operator notices; the service manager will surface the exit.
		return err
	}
	defer e.stopAll(context.Background())

	watcher, werr := fsnotify.NewWatcher()
	if werr == nil {
		defer func() { _ = watcher.Close() }()
		_ = watcher.Add(filepath.Dir(e.configPath))
		if filepath.Dir(e.configPath) != e.runtimeDir {
			_ = watcher.Add(e.runtimeDir)
		}
	} else {
		e.log.Warn("file watcher unavailable; relying on poll", "err", werr)
	}

	timer := time.NewTimer(0)
	defer timer.Stop()

	var events <-chan fsnotify.Event
	var errs <-chan error
	if watcher != nil {
		events = watcher.Events
		errs = watcher.Errors
	}

	for {
		select {
		case <-ctx.Done():
			e.log.Info("shutting down")
			return nil
		case ev := <-events:
			if ev.Name == e.configPath && ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
				e.reloadConfig(ctx)
			}
			// Any override change (or config change) re-evaluates soon.
			resetSoon(timer)
		case err := <-errs:
			e.log.Warn("watcher error", "err", err)
		case <-timer.C:
			next := e.cycle(ctx)
			timer.Reset(next)
		}
	}
}

// cycle performs one evaluation and returns the duration until the next wake.
func (e *Engine) cycle(ctx context.Context) time.Duration {
	now := time.Now()
	ov, err := schedule.LoadOverride(control.OverridePath(e.runtimeDir))
	if err != nil {
		e.log.Warn("read override", "err", err)
		ov = schedule.Override{}
	}
	desired := schedule.DesiredActive(e.cfg, ov, now)

	if desired {
		e.drive(ctx, now)
	} else if e.activeNow {
		e.stopAll(ctx)
		e.activeNow = false
		e.log.Info("deactivated (schedule/override)")
	}

	e.publish(ov, desired, now)

	if desired {
		return e.tickInterval()
	}
	return idlePoll
}

// drive marks the session active and pulses every activator once.
func (e *Engine) drive(ctx context.Context, now time.Time) {
	if !e.activeNow {
		e.activeNow = true
		e.log.Info("activated", "engine", string(e.cfg.Engine))
	}
	var firstErr string
	for _, a := range e.activators {
		if err := a.Tick(ctx); err != nil {
			e.log.Warn("activator tick failed", "activator", a.Name(), "err", err)
			if firstErr == "" {
				firstErr = a.Name() + ": " + err.Error()
			}
		}
	}
	e.lastErr = firstErr
	t := now
	e.lastTick = &t
}

// tickInterval returns the next active-mode sleep with jitter applied.
func (e *Engine) tickInterval() time.Duration {
	base := time.Duration(e.cfg.Input.IntervalSeconds) * time.Second
	if !e.cfg.UsesInput() {
		// Graph-only: the activator self-throttles to refresh_minutes, so a
		// coarse cadence is enough.
		base = idlePoll
	}
	j := e.cfg.Input.JitterSeconds
	if j > 0 {
		base += time.Duration(rand.Intn(2*j+1)-j) * time.Second
	}
	if base < time.Second {
		base = time.Second
	}
	return base
}

// publish writes the current status to disk.
func (e *Engine) publish(ov schedule.Override, desired bool, now time.Time) {
	names := make([]string, 0, len(e.activators))
	for _, a := range e.activators {
		names = append(names, a.Name())
	}
	st := control.Status{
		PID:           os.Getpid(),
		Engine:        string(e.cfg.Engine),
		Activators:    names,
		DesiredActive: desired,
		OverrideMode:  string(ov.EffectiveMode(now)),
		OverrideUntil: ov.Until,
		LastTick:      e.lastTick,
		LastError:     e.lastErr,
		UpdatedAt:     now,
		StartedAt:     e.startedAt,
	}
	if when, toActive, ok := schedule.NextChange(e.cfg, ov, now); ok {
		st.NextChange = &when
		st.NextActive = toActive
	}
	if err := control.WriteStatus(e.runtimeDir, st); err != nil {
		e.log.Warn("write status", "err", err)
	}
}

// reloadConfig re-reads config and rebuilds activators on success.
func (e *Engine) reloadConfig(ctx context.Context) {
	cfg, err := config.Load(e.configPath)
	if err != nil {
		e.log.Warn("config reload failed; keeping previous", "err", err)
		return
	}
	e.cfg = cfg
	e.stopAll(ctx)
	e.activeNow = false
	if err := e.rebuild(ctx); err != nil {
		e.log.Error("rebuild activators after reload failed", "err", err)
		return
	}
	e.log.Info("config reloaded", "engine", string(cfg.Engine))
}

// rebuild constructs the activator set for the current config.
func (e *Engine) rebuild(_ context.Context) error {
	if e.dryRun {
		e.activators = []activity.Activator{activity.NewDry(string(e.cfg.Engine), e.log)}
		return nil
	}
	var as []activity.Activator
	if e.cfg.UsesInput() {
		a, err := activity.NewInput(e.cfg.Input)
		if err != nil {
			return err
		}
		as = append(as, a)
	}
	if e.cfg.UsesGraph() {
		client, err := graph.New(e.cfg.Graph.TenantID, e.cfg.Graph.ClientID, e.tokenPath)
		if err != nil {
			return err
		}
		as = append(as, activity.NewGraph(client, e.cfg.Graph))
	}
	e.activators = as
	return nil
}

// stopAll reverts every activator (release assertions, clear presence).
func (e *Engine) stopAll(ctx context.Context) {
	for _, a := range e.activators {
		if err := a.Stop(ctx); err != nil {
			e.log.Warn("activator stop failed", "activator", a.Name(), "err", err)
		}
	}
}

func resetSoon(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(200 * time.Millisecond)
}
