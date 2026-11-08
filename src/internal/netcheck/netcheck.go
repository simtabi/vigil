// Package netcheck reports whether the machine currently has working internet
// access. It uses the standard captive-portal "generate_204" probes (a full
// DNS+TCP+HTTP check) with a short timeout, and a raw TCP dial as a fallback.
package netcheck

import (
	"context"
	"net"
	"net/http"
	"time"
)

// DefaultEndpoints are well-known endpoints that return HTTP 204 No Content when
// the network is unrestricted (used by OS captive-portal detection).
var DefaultEndpoints = []string{
	"http://cp.cloudflare.com/generate_204",
	"http://connectivitycheck.gstatic.com/generate_204",
}

// DefaultTCPAddr is dialed if every HTTP probe fails (e.g. HTTP egress blocked
// but raw TCP works). 1.1.1.1:443 is Cloudflare's anycast resolver.
const DefaultTCPAddr = "1.1.1.1:443"

// Checker performs connectivity checks. The zero value is not usable; call New.
type Checker struct {
	Endpoints []string
	TCPAddr   string
	Timeout   time.Duration
	Client    *http.Client
}

// New returns a Checker with sensible defaults (~3s timeout, no redirects).
func New() *Checker {
	return &Checker{Endpoints: DefaultEndpoints, TCPAddr: DefaultTCPAddr, Timeout: 3 * time.Second}
}

// Online reports whether any probe succeeds. It is best-effort: a single success
// returns true; all failures (or a cancelled ctx) return false.
func (c *Checker) Online(ctx context.Context) bool {
	to := c.Timeout
	if to <= 0 {
		to = 3 * time.Second
	}
	client := c.Client
	if client == nil {
		client = &http.Client{
			Timeout: to,
			// A captive portal answers with a redirect, not 204 — don't follow it.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}
	}
	for _, ep := range c.Endpoints {
		if probe(ctx, client, ep, to) {
			return true
		}
	}
	if c.TCPAddr != "" {
		cctx, cancel := context.WithTimeout(ctx, to)
		defer cancel()
		d := net.Dialer{}
		if conn, err := d.DialContext(cctx, "tcp", c.TCPAddr); err == nil {
			_ = conn.Close()
			return true
		}
	}
	return false
}

func probe(ctx context.Context, client *http.Client, url string, to time.Duration) bool {
	cctx, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusNoContent || resp.StatusCode/100 == 2
}

// Online is a package-level convenience using default settings.
func Online(ctx context.Context) bool { return New().Online(ctx) }
