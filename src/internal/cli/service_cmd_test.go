package cli

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/simtabi/vigil/internal/config"
)

func TestEnsureConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")

	created, err := ensureConfig(p)
	if err != nil || !created {
		t.Fatalf("first ensureConfig: created=%v err=%v", created, err)
	}
	if _, err := config.Load(p); err != nil {
		t.Fatalf("written config should be valid: %v", err)
	}

	created, err = ensureConfig(p)
	if err != nil || created {
		t.Fatalf("second ensureConfig should be a no-op: created=%v err=%v", created, err)
	}
}

func TestAlreadyInstalled(t *testing.T) {
	if !alreadyInstalled(errors.New("Init already exists")) {
		t.Error("should detect 'already exists'")
	}
	if !alreadyInstalled(errors.New("service vigil already installed")) {
		t.Error("should detect 'already installed'")
	}
	if alreadyInstalled(errors.New("permission denied")) {
		t.Error("false positive on unrelated error")
	}
}
