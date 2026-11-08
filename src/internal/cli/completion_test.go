package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompleteConfigKey(t *testing.T) {
	got, dir := completeConfigKey(nil, nil, "")
	if dir != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected NoFileComp, got %v", dir)
	}
	if !has(got, "engine") || !has(got, "timezone") || !has(got, "input.method") {
		t.Errorf("expected config keys, got %v", got)
	}
	// Once a key is present, no further key completion.
	if got, _ := completeConfigKey(nil, []string{"engine"}, ""); got != nil {
		t.Errorf("expected no completion after the key, got %v", got)
	}
}

func TestCompleteConfigSetValues(t *testing.T) {
	// engine values
	if got, _ := completeConfigSet(nil, []string{"engine"}, ""); !equalSet(got, []string{"input", "graph", "both"}) {
		t.Errorf("engine values = %v", got)
	}
	// booleans
	if got, _ := completeConfigSet(nil, []string{"schedule.enabled"}, ""); !equalSet(got, []string{"true", "false"}) {
		t.Errorf("bool values = %v", got)
	}
	// free-text key → no candidates
	if got, _ := completeConfigSet(nil, []string{"graph.client_id"}, ""); got != nil {
		t.Errorf("free-text key should have no candidates, got %v", got)
	}
	// timezone is prefix-filtered
	got, _ := completeConfigSet(nil, []string{"timezone"}, "Europe/Lo")
	if !has(got, "Europe/London") {
		t.Errorf("timezone prefix should include Europe/London, got %v", got)
	}
	for _, z := range got {
		if !strings.HasPrefix(strings.ToLower(z), "europe/lo") {
			t.Errorf("timezone candidate %q does not match the prefix", z)
		}
	}
}

func TestCompleteScope(t *testing.T) {
	got, dir := completeScope(nil, nil, "")
	if dir != cobra.ShellCompDirectiveNoFileComp || !equalSet(got, []string{"user", "system"}) {
		t.Errorf("scope completion = %v (%v)", got, dir)
	}
}

func has(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

func equalSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for _, w := range want {
		if !has(got, w) {
			return false
		}
	}
	return true
}
