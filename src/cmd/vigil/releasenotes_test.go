package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every release carries its CHANGELOG section as the body. release.yml hands
// GoReleaser that section with --release-notes, but GoReleaser reads the flag
// inside its changelog pipe, and `changelog: disable: true` skips the pipe
// whole: ssh-manager v3.2.0 and v3.4.0 were published with empty bodies that
// way, and the run reported success. So the config must not disable the
// changelog, and the workflow also sets the body itself after GoReleaser.
func TestReleasesCarryTheirChangelogSection(t *testing.T) {
	root := filepath.Join("..", "..")
	cfg, err := os.ReadFile(filepath.Join(root, ".goreleaser.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if regexp.MustCompile(`(?m)^changelog:\s*\n(?:[ \t]+#.*\n)*[ \t]+disable:\s*(true|"true")`).Match(cfg) {
		t.Error(".goreleaser.yaml sets changelog.disable: true, which makes GoReleaser ignore --release-notes")
	}
	wf, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--release-notes=", "gh release edit", "--notes-file"} {
		if !strings.Contains(string(wf), want) {
			t.Errorf("release.yml lacks %q: the release body must be set from the CHANGELOG section", want)
		}
	}
}

// Go releases are git only until the owner says otherwise, so release.yml must
// never start on its own: no push trigger of any kind, and a dispatch that
// refuses to publish without confirm=release-go. A tag push that started
// GoReleaser is how an unwanted release began elsewhere on 2026-11-02.
func TestReleaseRunsOnlyOnAConfirmedDispatch(t *testing.T) {
	wf, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if regexp.MustCompile(`(?m)^[ \t]*(push|schedule|pull_request|release)\s*:`).Match(wf) {
		t.Error("release.yml has an automatic trigger; it must run on workflow_dispatch only")
	}
	for _, want := range []string{"workflow_dispatch:", "confirm:", `"${CONFIRM}" != "release-go"`, "refs/tags/v[0-9]*"} {
		if !strings.Contains(string(wf), want) {
			t.Errorf("release.yml lacks %q: a release needs an explicit, confirmed dispatch on a version tag", want)
		}
	}
}
