// Package release_test guards how vigil is built and released. It holds tests
// only: the workflows are YAML and build/ is a directory, so there is no Go
// code here to test, just a policy to pin. The release-trigger and
// release-notes guards live beside the binary, in src/cmd/vigil.
package release_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repoRoot walks up from the test's directory to the repository root: the
// directory holding src/go.mod. The module lives in src/, so the root is its
// parent, not the directory holding go.mod.
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

func readFile(t *testing.T, parts ...string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// buildInputs are the files build/ may hold besides the OS folders: the tracked
// inputs and the checksums `make cross` writes.
var buildInputs = map[string]bool{"targets.txt": true, "nfpm.yaml": true, "checksums.txt": true}

// osFolders are the folders build/ may hold: one per OS, named by GOOS except
// darwin, which is macos. archives/ is written by `make dist`.
var osFolders = map[string]bool{
	"macos": true, "linux": true, "windows": true, "freebsd": true, "openbsd": true,
	"netbsd": true, "dragonfly": true, "solaris": true, "illumos": true, "aix": true,
	"plan9": true, "android": true, "ios": true, "js": true, "wasip1": true,
	"archives": true,
}

// build/ holds one folder per OS and nothing else: no build/dist/, no darwin/,
// no per-target subfolders and no tool intermediates. `make build`, `make cross`
// and `make dist` run this through `make check-build`, and CI runs it after
// building the snapshot binaries.
func TestBuildLayout(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "build")
	if problems := buildLayoutProblems(t, dir); len(problems) > 0 {
		t.Errorf("build/ breaks the layout (one folder per OS, see the Makefile):\n  %s", strings.Join(problems, "\n  "))
	}
}

// The layout check has to be able to fail: each of these trees breaks it.
func TestBuildLayoutRejectsStrayOutput(t *testing.T) {
	cases := map[string][]string{
		"a dist folder":        {"dist/vigil_linux_amd64"},
		"a darwin folder":      {"darwin/vigil_darwin_arm64"},
		"a per-target folder":  {"linux/vigil_linux_amd64/vigil"},
		"a loose binary":       {"vigil"},
		"goreleaser metadata":  {"artifacts.json"},
		"a nested macos build": {"macos/universal/vigil"},
	}
	for name, files := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range append([]string{"targets.txt"}, files...) {
				p := filepath.Join(dir, filepath.FromSlash(f))
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if len(buildLayoutProblems(t, dir)) == 0 {
				t.Errorf("%v passed the layout check", files)
			}
		})
	}
}

// A layout the build scripts really write passes, so the check is not simply
// refusing everything.
func TestBuildLayoutAcceptsTheOSFolders(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{
		"targets.txt", "nfpm.yaml", "checksums.txt",
		"macos/vigil", "macos/vigil_macos_arm64", "macos/vigil_macos_universal",
		"linux/vigil_linux_armv7", "windows/vigil_windows_amd64.exe",
		"archives/vigil_linux_amd64.tar.gz", "archives/checksums.txt",
	} {
		p := filepath.Join(dir, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if problems := buildLayoutProblems(t, dir); len(problems) > 0 {
		t.Errorf("a conforming build/ was refused:\n  %s", strings.Join(problems, "\n  "))
	}
}

func buildLayoutProblems(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "targets.txt")); err != nil {
		t.Fatalf("%s has no targets.txt: the check is not looking at build/", dir)
	}
	var problems []string
	for _, e := range entries {
		name := e.Name()
		switch {
		case name == ".DS_Store":
		case !e.IsDir() && buildInputs[name]:
		case e.IsDir() && osFolders[name]:
			inner, err := os.ReadDir(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			for _, f := range inner {
				if f.IsDir() {
					problems = append(problems, name+"/"+f.Name()+"/ is a subfolder; an OS folder holds binaries only")
				}
			}
		case e.IsDir():
			problems = append(problems, name+"/ is not an OS folder (darwin is written macos; there is no dist/)")
		default:
			problems = append(problems, name+" belongs in an OS folder, or nowhere")
		}
	}
	return problems
}

// skipDirs are never ours to police: git's own store, dependencies,
// generated output and local tool state.
var skipDirs = map[string]bool{
	".git": true, "vendor": true, "node_modules": true, "build": true, ".cache": true, ".claude": true,
}

// containerFile matches a file that exists only to build or run a container.
var containerFile = regexp.MustCompile(`(?i)^(dockerfile(\..*)?|.*\.dockerfile|\.dockerignore|containerfile|(docker-)?compose(\.[a-z0-9_-]+)?\.ya?ml)$`)

// containerInWorkflow matches a workflow step that builds, runs or publishes an image.
var containerInWorkflow = regexp.MustCompile(`(?m)uses:\s*['"]?docker/|\bdocker\s+(build|buildx|run|push|compose|login)\b|\bghcr\.io\b|^\s*dockers?(_v2)?\s*:`)

// vigil ships as a binary built from source. No container image is built or
// published, and nothing brings one back without this test changing.
func TestNoContainerImage(t *testing.T) {
	root := repoRoot(t)
	visited := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		visited++
		if containerFile.MatchString(d.Name()) {
			rel, _ := filepath.Rel(root, path)
			t.Errorf("%s builds or runs a container; vigil ships a binary built from source", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// The repository held 129 tracked files on 2026-11-07; a walk rooted in
	// the wrong place visits a handful.
	if visited < 80 {
		t.Fatalf("visited %d files, want at least 80: the walk is not looking where it should", visited)
	}

	workflows, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.y*ml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(workflows) < 3 {
		t.Fatalf("found %d workflow files, want at least 3 (build-binaries, ci, release)", len(workflows))
	}
	for _, f := range append(workflows, filepath.Join(root, "src", ".goreleaser.yaml")) {
		if m := containerInWorkflow.FindString(readFile(t, f)); m != "" {
			t.Errorf("%s: %q builds, runs or publishes an image", filepath.Base(f), strings.TrimSpace(m))
		}
	}
	if regexp.MustCompile(`package-ecosystem:\s*['"]?docker`).MatchString(readFile(t, root, ".github", "dependabot.yml")) {
		t.Error("dependabot.yml watches a docker ecosystem")
	}
}

// goInstall matches a `go install` of this module, by its github.com path or by
// the elided `…/cmd/vigil` form the docs used for it.
var goInstall = regexp.MustCompile("go install [^\\s`]*(github\\.com/|cmd/vigil)")

// Binaries ship on the GitHub release only, so the docs offer the release
// download or a build from source: no `go install` of this module, which would
// serve whatever the Go proxy cached for a tag rather than the released code.
func TestDocsOfferNoGoInstall(t *testing.T) {
	root := repoRoot(t)
	files := []string{filepath.Join(root, "README.md"), filepath.Join(root, "CONTRIBUTING.md")}
	docs := 0
	err := filepath.WalkDir(filepath.Join(root, "docs"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".md") {
			files = append(files, path)
			docs++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// docs/ held 12 pages on 2026-11-07.
	if docs < 8 {
		t.Fatalf("found %d docs pages, want at least 8: the walk is not looking at docs/", docs)
	}
	for _, f := range files {
		if m := goInstall.FindString(readFile(t, f)); m != "" {
			rel, _ := filepath.Rel(root, f)
			t.Errorf("%s offers %q; the docs offer the GitHub release or a build from source", rel, m)
		}
	}
}

// The pattern above has to be able to fail.
func TestGoInstallPatternMatches(t *testing.T) {
	for _, line := range []string{
		"go install github.com/simtabi/vigil/src/cmd/vigil@latest",
		"GOBIN=~/bin go install github.com/simtabi/vigil/cmd/vigil@v0.7.0",
		"re-run `go install …/cmd/vigil@latest`",
	} {
		if !goInstall.MatchString(line) {
			t.Errorf("%q is a go install of vigil and was not matched", line)
		}
	}
	for _, line := range []string{
		"there is no download, install-script, package-manager or `go install` channel",
		"go install golang.org/x/tools/cmd/stringer@latest", // another module
	} {
		if goInstall.MatchString(line) {
			t.Errorf("%q was matched, but offers no go install of vigil", line)
		}
	}
}

// The module is github.com/simtabi/vigil/src, so Go resolves a version through the src/vX.Y.Z tag
// and ignores vX.Y.Z. The shared simtabi/.github go-release.yml mirrors that alias onto the release
// tag's commit before GoReleaser runs, and never moves one that exists, as long as the caller
// leaves module-alias-prefix at its default, src/. The caller must also keep this repository's
// config, Go version file and changelog, run on macOS (the darwin targets need cgo and the macOS
// SDK) and under the release environment, as the workflow it replaced did.
func TestReleaseInputsMatchTheRepository(t *testing.T) {
	root := repoRoot(t)
	wf := readFile(t, root, ".github", "workflows", "release.yml")
	if !strings.Contains(wf, "uses: simtabi/.github/.github/workflows/go-release.yml@") {
		t.Fatal("release.yml does not call the shared go-release.yml; this guard reads its inputs")
	}
	defaults := map[string]string{
		"module-alias-prefix": "src/",
		"goreleaser-config":   "src/.goreleaser.yaml",
		"go-version-file":     "src/go.mod",
		"changelog":           "CHANGELOG.md",
	}
	required := map[string]string{
		"runs-on":     "macos-latest",
		"environment": "release",
	}
	input := func(name string) []string {
		return regexp.MustCompile(`(?m)^      ` + name + `:\s*['"]?([^'"\s#]*)`).FindStringSubmatch(wf)
	}
	for name, want := range defaults {
		if m := input(name); m != nil && m[1] != want {
			t.Errorf("release.yml sets %s to %q; this repository needs the shared default %q", name, m[1], want)
		}
	}
	for name, want := range required {
		if m := input(name); m == nil || m[1] != want {
			t.Errorf("release.yml must set %s: %s", name, want)
		}
	}

	cfg := readFile(t, root, "src", ".goreleaser.yaml")
	for _, want := range []string{"dist: .cache/goreleaser", "dir: src", "-X github.com/simtabi/vigil/src/internal/cli.version="} {
		if !strings.Contains(cfg, want) {
			t.Errorf("src/.goreleaser.yaml lacks %q", want)
		}
	}
	if strings.Contains(cfg, "dist: build") {
		t.Error("src/.goreleaser.yaml writes into build/, which holds one folder per OS and nothing else")
	}
}

// otherChannels are the GoReleaser blocks that publish somewhere other than the GitHub release:
// package managers, images and arbitrary publishers (~/.claude/rules/go.md, owner, 2026-11-13).
// nfpms stays: it builds the .deb/.rpm/.apk files that ship as release assets.
var otherChannels = []string{
	"brews", "homebrew_casks", "scoops", "winget", "chocolateys", "aurs", "aur_sources",
	"snapcrafts", "flatpaks", "nix", "dockers", "dockers_v2", "docker_manifests", "docker_signs",
	"kos", "krews", "npms", "cloudsmiths", "furies", "publishers", "blobs", "uploads", "artifactories",
}

// The GitHub release is the only channel. The Homebrew cask, Scoop manifest and Winget manifest
// this config published until 2026-11-13 are gone from it and must not come back; the shared
// workflow refuses such a config at release time, and this fails first, on the pull request.
func TestGoReleaserPublishesNowhereElse(t *testing.T) {
	cfg := readFile(t, repoRoot(t), "src", ".goreleaser.yaml")
	if !regexp.MustCompile(`(?m)^release:`).MatchString(cfg) {
		t.Fatal("src/.goreleaser.yaml has no release block; the GitHub release is the channel")
	}
	for _, k := range otherChannels {
		if regexp.MustCompile(`(?m)^` + k + `:`).MatchString(cfg) {
			t.Errorf("src/.goreleaser.yaml publishes through %q; binaries go to the GitHub release only", k)
		}
	}
}
