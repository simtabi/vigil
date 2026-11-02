package selfupdate

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/Masterminds/semver/v3"
)

// Relation is how the running version stands against the latest release.
type Relation int

const (
	// UpToDate means the running version is the release.
	UpToDate Relation = iota
	// ReleaseNewer means the release is strictly newer: a real update.
	ReleaseNewer
	// RunningAhead means installing the release would not move forward: the
	// running version is newer, is a source build between or past a tag, or
	// cannot be parsed at all.
	RunningAhead
)

func (r Relation) String() string {
	switch r {
	case UpToDate:
		return "up-to-date"
	case ReleaseNewer:
		return "release-newer"
	default:
		return "running-ahead"
	}
}

// ErrRunningNewer is returned when installing the latest release would replace
// the running binary with an older (or not provably newer) one.
var ErrRunningNewer = errors.New("the running version is newer than the latest release")

// sourceBuild matches the pre-release part `make build` stamps from git
// describe: "3-gabc1234", "3-gabc1234-dirty", or "dirty" on an exact tag.
var sourceBuild = regexp.MustCompile(`^(\d+-g[0-9a-f]+)?(-?dirty)?$`)

// isSourceBuild reports whether a parsed version is a git-describe build: N
// commits past a tag, a -dirty tree, or both. Such a build is at or past its
// base tag, never behind it.
func isSourceBuild(v *semver.Version) bool {
	pre := v.Prerelease()
	return pre != "" && sourceBuild.MatchString(pre)
}

// Compare decides how current stands against the latest release. Dev and
// snapshot builds (IsDev) and unparseable versions are always ahead. Plain semver
// sorts a git-describe version such as v0.7.0-3-gabc1234 as a pre-release
// below 0.7.0, which would offer the 0.7.0 binary as an update to a newer
// build; Compare counts such builds, and anything unparseable, as ahead of
// their base tag instead. Both the upgrade command and the TUI banner use it.
func Compare(current, latest string) Relation {
	if IsDev(current) {
		return RunningAhead // "", "dev", -dev, snapshot: never offered a release
	}
	cur, err := semver.NewVersion(current)
	if err != nil {
		return RunningAhead
	}
	rel, err := semver.NewVersion(latest)
	if err != nil {
		return UpToDate // nothing trustworthy to move to
	}
	if isSourceBuild(cur) {
		base, _ := cur.SetPrerelease("")
		base, _ = base.SetMetadata("")
		if rel.GreaterThan(&base) {
			return ReleaseNewer // a later tag than the one this was built from
		}
		return RunningAhead
	}
	switch {
	case rel.GreaterThan(cur):
		return ReleaseNewer
	case rel.Equal(cur):
		return UpToDate
	default:
		return RunningAhead
	}
}

// ShouldInstall is the downgrade guard: it reports whether the latest release
// may replace the running binary. A newer release is installed; the same
// version is not, even when forced; anything else is refused with
// ErrRunningNewer unless force is set.
func ShouldInstall(current, latest string, force bool) (bool, error) {
	switch Compare(current, latest) {
	case ReleaseNewer:
		return true, nil
	case UpToDate:
		return false, nil
	default:
		if force {
			return true, nil
		}
		return false, fmt.Errorf("%w: refusing to replace %s with %s (use --force to install it anyway)",
			ErrRunningNewer, displayVersion(current), latest)
	}
}

func displayVersion(v string) string {
	if v == "" {
		return "an unversioned build"
	}
	return v
}
