package cli

import (
	"io"
	"strings"
	"testing"
)

// runRoot resets persisted global-flag state and executes the root command with
// the given args, discarding output. cobra retains flag values across Execute
// calls, so the reset keeps tests independent.
func runRoot(args ...string) error {
	flagScope, flagConfig, flagFor, flagUntil = "user", "", "", ""
	flagJSON, flagVerbose, flagYes, flagNoInput, flagNoColor = false, false, false, false, false
	rootCmd.SilenceUsage, rootCmd.SilenceErrors = true, true
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func TestInvalidScopeIsRejected(t *testing.T) {
	err := runRoot("--scope", "bogus", "version")
	if err == nil || !strings.Contains(err.Error(), "invalid --scope") {
		t.Fatalf("expected an invalid --scope error, got %v", err)
	}
}

func TestValidScopeAccepted(t *testing.T) {
	if err := runRoot("--scope", "system", "version"); err != nil {
		t.Fatalf("system scope should be accepted: %v", err)
	}
}

func TestForMustBePositive(t *testing.T) {
	err := runRoot("on", "--for=-5m")
	if err == nil || !strings.Contains(err.Error(), "positive") {
		t.Fatalf("expected a positive-duration error, got %v", err)
	}
}

func TestForRejectsGarbage(t *testing.T) {
	err := runRoot("on", "--for=banana")
	if err == nil || !strings.Contains(err.Error(), "--for") {
		t.Fatalf("expected a --for parse error, got %v", err)
	}
}

func TestForAndUntilMutuallyExclusive(t *testing.T) {
	err := runRoot("off", "--for=1h", "--until=17:00")
	if err == nil || !strings.Contains(err.Error(), "only one of") {
		t.Fatalf("expected mutual-exclusion error, got %v", err)
	}
}
