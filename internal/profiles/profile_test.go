package profiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	in := Profile{
		OS:      []string{"macos"},
		Formats: []string{"pdf", "markdown"},
		Out:     "~/Documents/osaat",
		License: License{Mode: "best-effort", AgeRecipient: "age1xyz"},
		Insights: Insights{
			Forgotten:       true,
			ForgottenMonths: 6,
			AppleSilicon:    true,
		},
		Restore: Restore{Enabled: true},
	}
	path, err := Save("imani-mbp", in)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !strings.HasSuffix(path, ".toml") {
		t.Errorf("Save returned non-toml path: %q", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("profile file mode: got %o, want 600", perm)
	}

	out, err := Load("imani-mbp")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.Schema != SchemaProfileV1 {
		t.Errorf("schema lost: %q", out.Schema)
	}
	if out.License.Mode != "best-effort" || out.License.AgeRecipient != "age1xyz" {
		t.Errorf("license round-trip wrong: %+v", out.License)
	}
	if !out.Insights.AppleSilicon || out.Insights.ForgottenMonths != 6 {
		t.Errorf("insights round-trip wrong: %+v", out.Insights)
	}
}

func TestListReturnsSortedNames(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	for _, name := range []string{"work-mbp", "imani-mbp", "alpha"} {
		if _, err := Save(name, Profile{}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := List()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"alpha", "imani-mbp", "work-mbp"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestValidateNameRejectsTraversal(t *testing.T) {
	for _, bad := range []string{"", "..", ".", "../foo", "a/b", "a\\b", "x:y"} {
		if err := validateName(bad); err == nil {
			t.Errorf("validateName(%q) should error", bad)
		}
	}
	for _, ok := range []string{"imani-mbp", "work_machine", "alpha"} {
		if err := validateName(ok); err != nil {
			t.Errorf("validateName(%q) returned %v", ok, err)
		}
	}
}

func TestLoadMissingReturnsError(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	_, err := Load("nonexistent")
	if err == nil {
		t.Fatal("expected error loading missing profile")
	}
	if !strings.Contains(err.Error(), "read profile") {
		t.Errorf("error should mention 'read profile': %v", err)
	}
}

func TestListEmptyDirReturnsNil(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	got, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("List on empty/missing dir should return nil; got %v", got)
	}
	// Sanity: the profiles dir shouldn't have been created as a side effect.
	if _, err := os.Stat(filepath.Join(tmpHome, ".config", "osaat", "profiles")); !os.IsNotExist(err) {
		t.Errorf("List should not create the profiles dir")
	}
}
