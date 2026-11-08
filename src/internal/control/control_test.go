package control

import (
	"testing"
	"time"
)

func TestStatusRoundTrip(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().Round(time.Second)
	want := Status{
		PID:           4242,
		Engine:        "input",
		Activators:    []string{"input(test)"},
		DesiredActive: true,
		OverrideMode:  "on",
		UpdatedAt:     now,
		StartedAt:     now,
	}
	if err := WriteStatus(dir, want); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := ReadStatus(dir)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.PID != want.PID || got.Engine != want.Engine || got.DesiredActive != want.DesiredActive {
		t.Fatalf("round-trip mismatch: got %+v want %+v", got, want)
	}
	if len(got.Activators) != 1 || got.Activators[0] != "input(test)" {
		t.Fatalf("activators not preserved: %v", got.Activators)
	}
}

func TestReadStatus_Missing(t *testing.T) {
	if _, err := ReadStatus(t.TempDir()); err != ErrNoStatus {
		t.Fatalf("expected ErrNoStatus, got %v", err)
	}
}

func TestSingleInstanceLock(t *testing.T) {
	dir := t.TempDir()

	lock, ok, err := AcquireLock(dir)
	if err != nil || !ok {
		t.Fatalf("first lock should succeed: ok=%v err=%v", ok, err)
	}

	// A second acquisition in the same process on the same path must fail.
	_, ok2, err := AcquireLock(dir)
	if err != nil {
		t.Fatalf("second lock errored: %v", err)
	}
	if ok2 {
		t.Fatal("second lock should not be granted while the first is held")
	}

	if err := lock.Release(); err != nil {
		t.Fatalf("release: %v", err)
	}

	// After release, acquiring again should succeed.
	lock3, ok3, err := AcquireLock(dir)
	if err != nil || !ok3 {
		t.Fatalf("re-lock after release should succeed: ok=%v err=%v", ok3, err)
	}
	_ = lock3.Release()
}
