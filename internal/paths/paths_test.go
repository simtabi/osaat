package paths

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultDocumentsDirEndsInOsaat(t *testing.T) {
	d, err := DefaultDocumentsDir()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(d, string(os.PathSeparator)+"osaat") {
		t.Errorf("expected suffix .../osaat, got %q", d)
	}
}

func TestConfigDirAlwaysDotConfigOsaat(t *testing.T) {
	c, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(".config", "osaat")
	if !strings.HasSuffix(c, want) {
		t.Errorf("ConfigDir should end with %q; got %q", want, c)
	}
}

func TestProfilesAndLogsUnderConfig(t *testing.T) {
	cfg, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	p, _ := ProfilesDir()
	l, _ := LogsDir()
	if filepath.Dir(p) != cfg {
		t.Errorf("ProfilesDir parent: got %q, want %q", filepath.Dir(p), cfg)
	}
	if filepath.Dir(l) != cfg {
		t.Errorf("LogsDir parent: got %q, want %q", filepath.Dir(l), cfg)
	}
}

func TestTidyPathReplacesHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir on this runner")
	}
	in := filepath.Join(home, "Documents", "osaat", "report.json")
	got := TidyPath(in)
	if !strings.HasPrefix(got, "~") {
		t.Errorf("TidyPath should rewrite to ~; got %q", got)
	}
	other := "/etc/passwd"
	if got := TidyPath(other); got != other {
		t.Errorf("TidyPath should leave non-home paths alone; got %q", got)
	}
}

func TestScrubHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir on this runner")
	}
	in := "error reading " + filepath.Join(home, "Library", "Preferences", "foo.plist")
	got := ScrubHome(in)
	if strings.Contains(got, home) {
		t.Errorf("ScrubHome left home in output: %q", got)
	}
	if !strings.Contains(got, "~") {
		t.Errorf("ScrubHome should produce ~; got %q", got)
	}
}

func TestDefaultOutputDirIsDated(t *testing.T) {
	d, err := DefaultOutputDir()
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Base(d)
	if len(base) != 10 || base[4] != '-' || base[7] != '-' {
		t.Errorf("DefaultOutputDir last segment should be YYYY-MM-DD; got %q", base)
	}
}
