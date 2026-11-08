package activity

import (
	"context"
	"log/slog"
)

// dryActivator performs no real action; it only logs what it would do. Used by
// `vigil run --dry-run` to observe scheduling/engine behavior without injecting
// input, holding power assertions, or calling Graph.
type dryActivator struct {
	engine string
	log    *slog.Logger
}

// NewDry returns a no-op activator that logs its intended actions.
func NewDry(engine string, log *slog.Logger) Activator {
	return &dryActivator{engine: engine, log: log}
}

func (d *dryActivator) Name() string { return "dry(" + d.engine + ")" }

func (d *dryActivator) Tick(_ context.Context) error {
	d.log.Info("dry-run: would keep the session active", "engine", d.engine)
	return nil
}

func (d *dryActivator) Stop(_ context.Context) error {
	d.log.Info("dry-run: would revert (release assertions / clear presence)", "engine", d.engine)
	return nil
}
