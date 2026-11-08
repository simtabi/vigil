package netcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOnlineWhenProbeReturns204(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := &Checker{Endpoints: []string{ts.URL}, Timeout: 2 * time.Second, Client: ts.Client()}
	if !c.Online(context.Background()) {
		t.Fatal("expected online when the probe returns 204")
	}
}

func TestOfflineWhenAllProbesFail(t *testing.T) {
	// Closed/unreachable endpoints and TCP target → fast refusal, offline.
	c := &Checker{
		Endpoints: []string{"http://127.0.0.1:1/generate_204"},
		TCPAddr:   "127.0.0.1:1",
		Timeout:   1 * time.Second,
	}
	if c.Online(context.Background()) {
		t.Fatal("expected offline when every probe fails")
	}
}

func TestCaptivePortalRedirectIsNotOnline(t *testing.T) {
	// A portal responds with a redirect, not 204 → must read as offline.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Redirect(w, &http.Request{}, "http://portal.local/login", http.StatusFound)
	}))
	defer ts.Close()

	c := &Checker{Endpoints: []string{ts.URL}, TCPAddr: "", Timeout: 2 * time.Second, Client: ts.Client()}
	if c.Online(context.Background()) {
		t.Fatal("a 302 captive-portal response must not count as online")
	}
}
