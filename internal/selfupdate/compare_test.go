package selfupdate

import (
	"errors"
	"testing"
)

// TestCompare pins how a running version relates to the latest release. A
// source build between tags (git describe), a -dirty or -dev build, and any
// unparseable version count as ahead of the release, never behind it.
func TestCompare(t *testing.T) {
	cases := []struct {
		current, latest string
		want            Relation
	}{
		{"0.6.0", "0.7.0", ReleaseNewer},
		{"v0.6.0", "0.7.0", ReleaseNewer},
		{"0.7.0", "0.7.0", UpToDate},
		{"v0.7.0", "0.7.0", UpToDate},
		{"0.8.0", "0.7.0", RunningAhead},
		{"v0.7.0-3-gabc1234", "0.7.0", RunningAhead},
		{"v0.7.0-3-gabc1234-dirty", "0.7.0", RunningAhead},
		{"v0.7.0-dirty", "0.7.0", RunningAhead},
		{"v0.7.0-dev", "0.7.0", RunningAhead},
		{"0.0.0-dev+abc1234", "0.7.0", RunningAhead},
		{"dev", "0.7.0", RunningAhead},
		{"", "0.7.0", RunningAhead},
		{"not-a-version", "0.7.0", RunningAhead},
		// A later tag is still a real update for a build of an earlier tag.
		{"v0.7.0-3-gabc1234", "0.8.0", ReleaseNewer},
		// An ordinary pre-release is ordered by semver.
		{"0.8.0-rc.1", "0.8.0", ReleaseNewer},
		{"0.8.0-rc.1", "0.7.0", RunningAhead},
	}
	for _, tc := range cases {
		if got := Compare(tc.current, tc.latest); got != tc.want {
			t.Errorf("Compare(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.want)
		}
	}
}

// TestShouldInstall is the downgrade guard both upgrade paths go through: an
// older release is refused unless forced, a newer one is installed, and the
// same version is a no-op even when forced.
func TestShouldInstall(t *testing.T) {
	cases := []struct {
		name            string
		current, latest string
		force           bool
		install         bool
		refused         bool
	}{
		{"release older than running tag", "0.8.0", "0.7.0", false, false, true},
		{"git-describe build ahead of tag", "v0.7.0-3-gabc1234", "0.7.0", false, false, true},
		{"dirty build", "v0.7.0-3-gabc1234-dirty", "0.7.0", false, false, true},
		{"dev build", "v0.7.0-dev", "0.7.0", false, false, true},
		{"unparseable version", "dev", "0.7.0", false, false, true},
		{"release newer", "0.6.0", "0.7.0", false, true, false},
		{"same version", "v0.7.0", "0.7.0", false, false, false},
		{"same version forced", "v0.7.0", "0.7.0", true, false, false},
		{"force overrides older release", "0.8.0", "0.7.0", true, true, false},
		{"force overrides source build", "v0.7.0-3-gabc1234", "0.7.0", true, true, false},
		{"force overrides unparseable", "dev", "0.7.0", true, true, false},
	}
	for _, tc := range cases {
		install, err := ShouldInstall(tc.current, tc.latest, tc.force)
		if install != tc.install {
			t.Errorf("%s: ShouldInstall(%q, %q, %v) install = %v, want %v",
				tc.name, tc.current, tc.latest, tc.force, install, tc.install)
		}
		if refused := errors.Is(err, ErrRunningNewer); refused != tc.refused {
			t.Errorf("%s: refused = %v (err %v), want %v", tc.name, refused, err, tc.refused)
		}
		if err != nil && !errors.Is(err, ErrRunningNewer) {
			t.Errorf("%s: unexpected error %v", tc.name, err)
		}
	}
}

// TestCheckInfoBanner pins what Check reports, which is what the TUI banner
// reads: no "update available" for a build ahead of the release.
func TestCheckInfoBanner(t *testing.T) {
	cases := []struct {
		current          string
		available, ahead bool
	}{
		{"0.6.0", true, false},
		{"v0.7.0", false, false},
		{"v0.7.0-3-gabc1234", false, true},
		{"v0.7.0-3-gabc1234-dirty", false, true},
		{"dev", false, true},
	}
	for _, tc := range cases {
		info := checkInfo(tc.current, "0.7.0", "")
		if info.Available != tc.available || info.Ahead != tc.ahead {
			t.Errorf("checkInfo(%q) = available %v ahead %v, want %v %v",
				tc.current, info.Available, info.Ahead, tc.available, tc.ahead)
		}
	}
}
