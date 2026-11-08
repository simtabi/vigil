package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repoRoot walks up from the test's directory to the repository root: the
// directory holding src/go.mod. The module lives in src/, so the root is its
// parent, not the directory holding go.mod, and a fixed "../.." from here now
// lands in src/.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "src", "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("src/go.mod not found above the test directory")
		}
		dir = parent
	}
}

// Every release carries its CHANGELOG section as the body. release.yml hands
// GoReleaser that section with --release-notes, but GoReleaser reads the flag
// inside its changelog pipe, and `changelog: disable: true` skips the pipe
// whole: ssh-manager v3.2.0 and v3.4.0 were published with empty bodies that
// way, and the run reported success. So the config must not disable the
// changelog, and the workflow also sets the body itself after GoReleaser.
func TestReleasesCarryTheirChangelogSection(t *testing.T) {
	root := repoRoot(t)
	cfg, err := os.ReadFile(filepath.Join(root, "src", ".goreleaser.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if regexp.MustCompile(`(?m)^changelog:\s*\n(?:[ \t]+#.*\n)*[ \t]+disable:\s*(true|"true")`).Match(cfg) {
		t.Error("src/.goreleaser.yaml sets changelog.disable: true, which makes GoReleaser ignore --release-notes")
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

// automaticTrigger matches a trigger that starts a workflow without a person
// dispatching it, at any indentation under `on:`.
var automaticTrigger = regexp.MustCompile(`(?m)^[ \t]*(push|schedule|pull_request|pull_request_target|release|workflow_run)\s*:`)

// Go releases are git only until the owner says otherwise, so release.yml must
// never start on its own: no push, schedule, pull_request or release trigger,
// and a dispatch whose first step refuses to publish without confirm=release-go
// on a vX.Y.Z tag. A tag push that started GoReleaser is how an unwanted
// release began elsewhere on 2026-11-02.
func TestReleaseRunsOnlyOnAConfirmedDispatch(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	wf := string(raw)
	if m := automaticTrigger.FindString(wf); m != "" {
		t.Errorf("release.yml has an automatic trigger (%q); it must run on workflow_dispatch only", strings.TrimSpace(m))
	}
	for _, want := range []string{"workflow_dispatch:", "confirm:", "required: true", `"${CONFIRM}" != "release-go"`, "refs/tags/v[0-9]*"} {
		if !strings.Contains(wf, want) {
			t.Errorf("release.yml lacks %q: a release needs an explicit, confirmed dispatch on a version tag", want)
		}
	}

	// The refusal must be the first step, before anything is checked out,
	// built or published.
	_, steps, ok := strings.Cut(wf, "steps:")
	if !ok {
		t.Fatal("release.yml has no steps")
	}
	first := steps
	if items := regexp.MustCompile(`(?m)^\s*- `).FindAllStringIndex(steps, 2); len(items) == 2 {
		first = steps[:items[1][0]]
	}
	for _, want := range []string{`"release-go"`, "refs/tags/v[0-9]*", "exit 1"} {
		if !strings.Contains(first, want) {
			t.Errorf("the first release step must refuse without confirm=release-go on a vX.Y.Z tag; it lacks %q:\n%s", want, first)
		}
	}
}
