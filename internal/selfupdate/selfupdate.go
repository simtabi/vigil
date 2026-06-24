// Package selfupdate wraps creativeprojects/go-selfupdate to update the mta
// binary from its GitHub releases, with checksum validation. Channel detection
// and service stop/restart orchestration live with the caller (internal/cli).
package selfupdate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/creativeprojects/go-selfupdate"
)

// Slug is the GitHub owner/repo releases are pulled from.
const Slug = "simtabi/ms-teams-activity"

// ErrDevVersion is returned when self-update is attempted on a dev build.
var ErrDevVersion = errors.New("this is a development build; self-update only works on released versions")

// Info summarizes a version check or applied update.
type Info struct {
	Current   string
	Latest    string
	Available bool
	Notes     string
}

// IsDev reports whether a version string is an unreleased build (empty, "dev",
// or any snapshot/dev-tagged version like "0.0.0-dev+<sha>"). Self-update is
// refused for these.
func IsDev(v string) bool {
	return v == "" || strings.Contains(v, "dev") || strings.Contains(v, "snapshot")
}

func repository() selfupdate.Repository { return selfupdate.ParseSlug(Slug) }

func newUpdater() (*selfupdate.Updater, error) {
	return selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
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
	info := Info{Current: current, Latest: rel.Version(), Notes: rel.ReleaseNotes}
	if !IsDev(current) {
		info.Available = rel.GreaterThan(current)
	}
	return info, nil
}

// Apply downloads the latest release (if newer) and replaces this binary,
// verifying the checksum. The caller must stop a running service first.
func Apply(ctx context.Context, current string) (Info, error) {
	if IsDev(current) {
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
	rel, err := up.UpdateCommand(ctx, exe, current, repository())
	if err != nil {
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
