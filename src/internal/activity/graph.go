package activity

import (
	"context"
	"sync"
	"time"

	"github.com/simtabi/vigil/internal/config"
	"github.com/simtabi/vigil/internal/graph"
)

// graphActivator keeps a sticky preferred presence via Microsoft Graph. It
// re-asserts presence on the configured refresh cadence (not every tick) and
// clears the preferred presence on Stop so the account reverts to automatic
// presence behavior.
type graphActivator struct {
	client  *graph.Client
	cfg     config.GraphConfig
	refresh time.Duration

	mu      sync.Mutex
	lastSet time.Time
	everSet bool
}

// NewGraph builds the Graph activator from a configured client and settings.
func NewGraph(client *graph.Client, cfg config.GraphConfig) Activator {
	return &graphActivator{
		client:  client,
		cfg:     cfg,
		refresh: time.Duration(cfg.RefreshMinutes) * time.Minute,
	}
}

func (g *graphActivator) Name() string { return "graph(preferred-presence)" }

func (g *graphActivator) Tick(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.everSet && time.Since(g.lastSet) < g.refresh {
		return nil
	}
	if err := g.client.SetPreferredPresence(ctx, g.cfg.Availability, g.cfg.Activity, g.cfg.Expiration); err != nil {
		return err
	}
	g.lastSet = time.Now()
	g.everSet = true
	return nil
}

func (g *graphActivator) Stop(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.everSet {
		return nil
	}
	g.everSet = false
	return g.client.ClearPreferredPresence(ctx)
}
