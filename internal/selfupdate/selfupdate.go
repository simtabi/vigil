// Package selfupdate wraps creativeprojects/go-selfupdate to update the vigil
// binary from its GitHub releases, with checksum validation. Channel detection
// and service stop/restart orchestration live with the caller (internal/cli).
package selfupdate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/creativeprojects/go-selfupdate"
)

// Slug is the GitHub owner/repo releases are pulled from.
const Slug = "simtabi/vigil"

// ErrDevVersion is returned when self-update is attempted on a dev build.
var ErrDevVersion = errors.New("this is a development build; self-update only works on released versions")

// Info summarizes a version check or applied update.
type Info struct {
	Current   string
	Latest    string
	Available bool
	// Ahead is set when the running version is newer than the release, is a
	// source build past its tag, or cannot be parsed: installing the release
	// would not move forward, so it is refused unless forced.
	Ahead bool
	Notes string
}

// IsDev reports whether a version string is an unreleased build (empty, "dev",
// or any snapshot/dev-tagged version like "0.0.0-dev+<sha>"). Self-update is
// refused for these.
func IsDev(v string) bool {
	return v == "" || strings.Contains(v, "dev") || strings.Contains(v, "snapshot")
}

func repository() selfupdate.Repository { return selfupdate.ParseSlug(Slug) }

func newUpdater() (*selfupdate.Updater, error) {
	cfg := selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	}
	// Assets use the friendly "macos" OS token (not Go's "darwin"), so tell the
	// updater to match macos_<arch> assets when running on a Mac.
	if runtime.GOOS == "darwin" {
		cfg.OS = "macos"
	}
	return selfupdate.NewUpdater(cfg)
}

// Check reports whether a newer release exists. It does not modify anything.
func Check(ctx context.Context, current string) (Info, error) {
	up, err := newUpdater()
	if err != nil {
		return Info{}, err
	}
	rel, found, err := up.DetectLatest(ctx, repository())
	if err != nil {
		return Info{}, err
	}
	if !found || rel == nil {
		return Info{Current: current}, nil
	}
	return checkInfo(current, rel.Version(), rel.ReleaseNotes), nil
}

// checkInfo builds Check's answer from Compare. Available, which drives the TUI
// "update available" banner, is set only for a strictly newer release.
func checkInfo(current, latest, notes string) Info {
	info := Info{Current: current, Latest: latest, Notes: notes}
	switch Compare(current, latest) {
	case ReleaseNewer:
		info.Available = true
	case RunningAhead:
		info.Ahead = true
	}
	return info
}

// Apply downloads the latest release (if newer) and replaces this binary,
// verifying the checksum. The caller must stop a running service first. It is
// ApplyWith without force.
func Apply(ctx context.Context, current string) (Info, error) {
	return ApplyWith(ctx, current, false)
}

// ApplyWith is Apply with an explicit force. Without force it never installs a
// release older than (or not provably newer than) the running version: that
// returns ErrRunningNewer, and a dev build returns ErrDevVersion. With force it
// installs the latest release over either, but never reinstalls the same one.
func ApplyWith(ctx context.Context, current string, force bool) (Info, error) {
	if IsDev(current) && !force {
		return Info{}, ErrDevVersion
	}
	exe, err := ExecutablePath()
	if err != nil {
		return Info{}, err
	}
	up, err := newUpdater()
	if err != nil {
		return Info{}, err
	}
	rel, found, err := up.DetectLatest(ctx, repository())
	if err != nil {
		return Info{}, err
	}
	if !found || rel == nil {
		return Info{Current: current}, nil
	}
	// Only replace the binary when the release is strictly newer — never downgrade
	// to a re-tagged or yanked "latest", or below a source build — unless forced.
	install, err := ShouldInstall(current, rel.Version(), force)
	if err != nil {
		return Info{Current: current, Latest: rel.Version(), Ahead: true}, err
	}
	if !install {
		return Info{Current: current, Latest: rel.Version()}, nil
	}
	if err := up.UpdateTo(ctx, rel, exe); err != nil {
		return Info{}, err
	}
	return Info{Current: current, Latest: rel.Version(), Available: true, Notes: rel.ReleaseNotes}, nil
}

// ExecutablePath returns the resolved (symlink-followed) path of this binary,
// which is the file that will be replaced on update.
func ExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved, nil
	}
	return exe, nil
}
