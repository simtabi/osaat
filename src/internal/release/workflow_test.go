// Package release_test guards how osaat is built and released. It holds tests
// only: the workflows are YAML and build/ is a directory, so there is no Go code
// here to test, just a policy to pin.
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

// The trigger guard itself is src/cmd/osaat's TestReleaseRunsOnlyOnAConfirmedDispatch.
// This one adds the order: the refusal must be the first step, before anything
// is checked out, built or published.
func TestReleaseRefusesFirst(t *testing.T) {
	wf := readFile(t, repoRoot(t), ".github", "workflows", "release.yml")
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
	// The module is github.com/simtabi/osaat/src, which Go resolves only
	// through a src/vX.Y.Z tag, so the release has to create it.
	if !strings.Contains(wf, `alias_tag="src/${GITHUB_REF_NAME}"`) {
		t.Error("release.yml does not mirror the src/vX.Y.Z tag Go resolves the module through")
	}
}

// buildInputs are the files build/ may hold besides the OS folders.
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
// no per-target subfolders and no tool intermediates. `make check-build` runs
// this after every build, and CI runs it after `make cross`.
func TestBuildLayout(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "build")
	if problems := buildLayoutProblems(t, dir); len(problems) > 0 {
		t.Errorf("build/ breaks the layout (one folder per OS, see the Makefile):\n  %s", strings.Join(problems, "\n  "))
	}
}

// The layout check has to be able to fail: each of these trees breaks it.
func TestBuildLayoutRejectsStrayOutput(t *testing.T) {
	cases := map[string][]string{
		"a dist folder":        {"dist/osaat_linux_amd64"},
		"a darwin folder":      {"darwin/osaat_darwin_arm64"},
		"a per-target folder":  {"linux/osaat_linux_amd64/osaat"},
		"a loose binary":       {"osaat"},
		"goreleaser metadata":  {"artifacts.json"},
		"a nested macos build": {"macos/universal/osaat"},
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

// And it has to pass the layout it describes, or it rejects everything.
func TestBuildLayoutAcceptsTheLayout(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{
		"targets.txt", "checksums.txt",
		"macos/osaat_macos_arm64", "macos/osaat_macos_universal",
		"linux/osaat_linux_armv7", "windows/osaat_windows_amd64.exe",
		"freebsd/osaat_freebsd_amd64", "archives/osaat_linux_amd64.tar.gz",
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
		t.Errorf("a conforming build/ was rejected:\n  %s", strings.Join(problems, "\n  "))
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

// skipDirs are never ours to police: git's own store, dependencies, generated
// output (build/, GoReleaser's .cache/, the pre-src/ layout's bin/ and dist/)
// and local tool state.
var skipDirs = map[string]bool{
	".git": true, "vendor": true, "node_modules": true, "build": true, ".cache": true,
	".claude": true, "bin": true, "dist": true,
}

// containerFile matches a file that exists only to build or run a container.
var containerFile = regexp.MustCompile(`(?i)^(dockerfile(\..*)?|.*\.dockerfile|\.dockerignore|containerfile|(docker-)?compose(\.[a-z0-9_-]+)?\.ya?ml)$`)

// containerInWorkflow matches a workflow step that builds, runs or publishes an image.
var containerInWorkflow = regexp.MustCompile(`(?m)uses:\s*['"]?docker/|\bdocker\s+(build|buildx|run|push|compose|login)\b|\bghcr\.io\b|^\s*dockers?(_v2)?\s*:`)

// osaat ships as a binary built from source. No container image is built or
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
			t.Errorf("%s builds or runs a container; osaat ships a binary built from source", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// The repository held 122 tracked files on 2026-11-07; a walk rooted in
	// the wrong place visits a handful.
	if visited < 80 {
		t.Fatalf("visited %d files, want at least 80: the walk is not looking where it should", visited)
	}

	workflows, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.y*ml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(workflows) < 2 {
		t.Fatalf("found %d workflow files, want at least 2 (ci, release)", len(workflows))
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

// Releases are on hold, so the docs offer building from source only: no
// `go install` of this module, which would also serve whatever the Go proxy
// cached for a tag rather than the code in the checkout.
func TestDocsOfferNoGoInstall(t *testing.T) {
	root := repoRoot(t)
	files := []string{filepath.Join(root, "README.md"), filepath.Join(root, "CONTRIBUTING.md")}
	for _, pattern := range []string{"*.md", filepath.Join("*", "*.md")} {
		docs, err := filepath.Glob(filepath.Join(root, "docs", pattern))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, docs...)
	}
	// docs/ held 5 top-level pages and 6 under tools/ on 2026-11-07.
	if len(files) < 10 {
		t.Fatalf("found %d documentation files, want at least 10: the glob is not looking at docs/", len(files))
	}
	goInstall := regexp.MustCompile("go install [^\\s`]*github\\.com/")
	for _, f := range files {
		if m := goInstall.FindString(readFile(t, f)); m != "" {
			rel, _ := filepath.Rel(root, f)
			t.Errorf("%s offers %q; while releases are on hold the docs build from source", rel, m)
		}
	}
}
