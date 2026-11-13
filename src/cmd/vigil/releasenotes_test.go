package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
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

// Every release carries its CHANGELOG section as the body. The shared simtabi/.github
// go-release.yml hands GoReleaser that section with --release-notes, then reads the published body
// back and fails the run if it is shorter than the section; it replaced this workflow's own
// `gh release edit --notes-file` repair on 2026-11-13. GoReleaser reads the flag inside its
// changelog pipe, and `changelog: disable: true` skips the pipe whole: ssh-manager v3.2.0 and
// v3.4.0 were published with empty bodies that way, and the run reported success. So the config
// must not disable the changelog, and the caller must not point the notes at another file.
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
	if !strings.Contains(string(wf), "uses: simtabi/.github/.github/workflows/go-release.yml@") {
		t.Error("release.yml does not call the shared go-release.yml, which sets and verifies the body")
	}
	if m := regexp.MustCompile(`(?m)^\s+changelog:\s*(\S+)`).FindStringSubmatch(string(wf)); m != nil && m[1] != "CHANGELOG.md" {
		t.Errorf("release.yml takes the notes from %s; they come from CHANGELOG.md", m[1])
	}
	if _, err := os.Stat(filepath.Join(root, "CHANGELOG.md")); err != nil {
		t.Error("CHANGELOG.md is missing; the release notes come from it")
	}
}

// release.yml publishes a release on a pushed version tag, or on request for a tag already
// pushed, and does nothing else: its triggers are exactly those two, and its one job calls the
// shared simtabi/.github go-release.yml (owner, 2026-11-13). A branch push, a schedule, a pull
// request or a release event must never start it. It replaces the dispatch-only, confirm=release-go
// guard of 2026-11-02.
//
// This reads the workflow's indentation, two spaces per level, as the file is written. A
// reformatted file fails here rather than passing unread.
func TestReleaseRunsOnAVersionTagOrOnRequest(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	wf := string(raw)

	on := yamlBlock(wf, "", "on")
	if on == "" {
		t.Fatal("release.yml has no top-level on: block")
	}
	if got := strings.Join(yamlKeys(on, "  "), ","); got != "push,workflow_dispatch" {
		t.Fatalf("release.yml triggers [%s], want exactly push (version tags) and workflow_dispatch", got)
	}

	push := yamlBlock(on, "  ", "push")
	if got := strings.Join(yamlKeys(push, "    "), ","); got != "tags" {
		t.Fatalf("release.yml's push filters [%s], want tags only, never a branch", got)
	}
	tags := regexp.MustCompile(`(?m)^      - '([^']*)'\s*$`).FindAllStringSubmatch(yamlBlock(push, "    ", "tags"), -1)
	if len(tags) == 0 {
		t.Fatal("release.yml's push has no quoted tag filters")
	}
	for _, m := range tags {
		if !strings.HasPrefix(m[1], "v[0-9]+.[0-9]+.[0-9]+") {
			t.Errorf("tag filter %q is not a version tag", m[1])
		}
	}

	tagInput := yamlBlock(yamlBlock(yamlBlock(on, "  ", "workflow_dispatch"), "    ", "inputs"), "      ", "tag")
	if !regexp.MustCompile(`(?m)^        required: true\s*$`).MatchString(tagInput) {
		t.Fatal("release.yml has no required tag input on workflow_dispatch")
	}

	jobs := yamlBlock(wf, "", "jobs")
	if got := strings.Join(yamlKeys(jobs, "  "), ","); got != "release" {
		t.Fatalf("release.yml jobs [%s], want the one release job", got)
	}
	job := yamlBlock(jobs, "  ", "release")
	if !regexp.MustCompile(`(?m)^    uses: simtabi/\.github/\.github/workflows/go-release\.yml@[0-9a-f]{40}\b`).MatchString(job) {
		t.Fatalf("the release job must call the shared go-release.yml pinned to a commit:\n%s", job)
	}
	if !strings.Contains(job, "      tag: ${{ inputs.tag || github.ref_name }}\n") {
		t.Error("the release job must pass tag: ${{ inputs.tag || github.ref_name }}")
	}
}

// yamlBlock returns the lines nested under the key at the given indent, or "" when it is absent.
func yamlBlock(text, indent, key string) string {
	var out []string
	in := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, indent+key+":") {
			in = true
			continue
		}
		if in {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(line, indent+" ") {
				break
			}
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// yamlKeys lists the mapping keys at exactly the given indent, sorted.
func yamlKeys(text, indent string) []string {
	re := regexp.MustCompile(`^` + indent + `([A-Za-z_][A-Za-z0-9_-]*):`)
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if m := re.FindStringSubmatch(line); m != nil {
			out = append(out, m[1])
		}
	}
	sort.Strings(out)
	return out
}
