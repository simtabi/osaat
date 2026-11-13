package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// release.yml publishes a release on a pushed version tag, or on request for a tag already
// pushed, and does nothing else: its triggers are exactly those two, and its one job calls the
// shared simtabi/.github go-release.yml (owner, 2026-11-13). A branch push, a schedule, a pull
// request or a release event must never start it. It replaces the dispatch-only, confirm=release-go
// guard of 2026-11-02.
//
// The module has no YAML dependency, so this reads the workflow's indentation: two spaces per
// level, as the file is written. A reformatted file fails here rather than passing unread.
func TestReleaseRunsOnAVersionTagOrOnRequest(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	wf := string(raw)

	on := block(wf, "", "on")
	if on == "" {
		t.Fatal("release.yml has no top-level on: block")
	}
	if got := strings.Join(keys(on, "  "), ","); got != "push,workflow_dispatch" {
		t.Fatalf("release.yml triggers [%s], want exactly push (version tags) and workflow_dispatch", got)
	}

	push := block(on, "  ", "push")
	if got := strings.Join(keys(push, "    "), ","); got != "tags" {
		t.Fatalf("release.yml's push filters [%s], want tags only, never a branch", got)
	}
	tags := regexp.MustCompile(`(?m)^      - '([^']*)'\s*$`).FindAllStringSubmatch(block(push, "    ", "tags"), -1)
	if len(tags) == 0 {
		t.Fatal("release.yml's push has no quoted tag filters")
	}
	for _, m := range tags {
		if !strings.HasPrefix(m[1], "v[0-9]+.[0-9]+.[0-9]+") {
			t.Errorf("tag filter %q is not a version tag", m[1])
		}
	}

	tagInput := block(block(block(on, "  ", "workflow_dispatch"), "    ", "inputs"), "      ", "tag")
	if !regexp.MustCompile(`(?m)^        required: true\s*$`).MatchString(tagInput) {
		t.Fatal("release.yml has no required tag input on workflow_dispatch")
	}

	jobs := block(wf, "", "jobs")
	if got := strings.Join(keys(jobs, "  "), ","); got != "release" {
		t.Fatalf("release.yml jobs [%s], want the one release job", got)
	}
	job := block(jobs, "  ", "release")
	if !regexp.MustCompile(`(?m)^    uses: simtabi/\.github/\.github/workflows/go-release\.yml@[0-9a-f]{40}\b`).MatchString(job) {
		t.Fatalf("the release job must call the shared go-release.yml pinned to a commit:\n%s", job)
	}
	if !strings.Contains(job, "      tag: ${{ inputs.tag || github.ref_name }}\n") {
		t.Error("the release job must pass tag: ${{ inputs.tag || github.ref_name }}")
	}
}

// block returns the lines nested under the key at the given indent, or "" when it is absent.
func block(text, indent, key string) string {
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

// keys lists the mapping keys at exactly the given indent.
func keys(text, indent string) []string {
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
