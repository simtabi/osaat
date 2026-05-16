package linux

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simtabi/osaat/internal/audit"
)

func TestParseSnapOutputWithHeader(t *testing.T) {
	out := []byte("" +
		"Name      Version       Rev    Tracking       Publisher  Notes\n" +
		"core22    20240408      1380   latest/stable  canonical  base\n" +
		"firefox   124.0.2-1     4173   latest/stable  mozilla    -\n" +
		"helm      3.14.4        372    latest/stable  snapcrafters  classic\n" +
		"")
	got := parseSnapOutput(out)
	if len(got) != 3 {
		t.Fatalf("want 3 records; got %d: %+v", len(got), got)
	}
	if got[0].Name != "core22" || got[0].Version != "20240408" {
		t.Errorf("core22 record: %+v", got[0])
	}
	if got[0].Author != "canonical" {
		t.Errorf("publisher: got %q", got[0].Author)
	}
	if got[1].ReinstallCmd != "snap install firefox" {
		t.Errorf("firefox reinstall: %q", got[1].ReinstallCmd)
	}
	if got[0].Source != audit.SourceSnap {
		t.Errorf("source: %q", got[0].Source)
	}
}

func TestParseSnapStripsVerifiedMark(t *testing.T) {
	out := []byte("" +
		"Name    Version  Rev  Tracking       Publisher   Notes\n" +
		"vlc     3.0.20   3262 latest/stable  videolan✓   -\n" +
		"")
	got := parseSnapOutput(out)
	if len(got) != 1 || got[0].Author != "videolan" {
		t.Errorf("✓ should be stripped: %+v", got)
	}
}

func TestSnapReinstallNonStableChannel(t *testing.T) {
	out := []byte("" +
		"Name    Version  Rev  Tracking         Publisher  Notes\n" +
		"node    20.12.2  150  20/stable        nodejs     classic\n" +
		"")
	got := parseSnapOutput(out)
	if got[0].ReinstallCmd != "snap install node --channel=20/stable" {
		t.Errorf("channel-aware reinstall: %q", got[0].ReinstallCmd)
	}
}

func TestParseFlatpakOutput(t *testing.T) {
	out := []byte("" +
		"org.mozilla.firefox\tFirefox\t124.0\tstable\tflathub\tsystem\n" +
		"com.github.tchx84.Flatseal\tFlatseal\t2.2.0\tstable\tflathub\tsystem\n" +
		"")
	got := parseFlatpakOutput(out)
	if len(got) != 2 {
		t.Fatalf("want 2 records; got %d", len(got))
	}
	if got[0].PkgID != "org.mozilla.firefox" || got[0].Name != "Firefox" {
		t.Errorf("firefox record: %+v", got[0])
	}
	if got[0].ReinstallCmd != "flatpak install -y flathub org.mozilla.firefox" {
		t.Errorf("reinstall: %q", got[0].ReinstallCmd)
	}
	if got[0].Source != audit.SourceFlatpak {
		t.Errorf("source: %q", got[0].Source)
	}
}

func TestParseFlatpakHandlesNonStableBranch(t *testing.T) {
	out := []byte("org.gnome.Shell\tGNOME Shell\t46.0\tnightly\tgnome-nightly\tuser\n")
	got := parseFlatpakOutput(out)
	if got[0].ReinstallCmd != "flatpak install -y gnome-nightly org.gnome.Shell//nightly" {
		t.Errorf("branch-suffixed reinstall: %q", got[0].ReinstallCmd)
	}
}

func TestParseFlatpakSkipsBlankAndNonID(t *testing.T) {
	out := []byte("" +
		"\n" +
		"NoDotName\tX\t1.0\tstable\tflathub\tsystem\n" + // no dot in first field
		"org.example.app\tExample\t1.0\tstable\tflathub\tsystem\n" +
		"")
	got := parseFlatpakOutput(out)
	if len(got) != 1 || got[0].PkgID != "org.example.app" {
		t.Errorf("got %+v", got)
	}
}

func TestAppImageWalkPicksUpOnlyAppImageFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"Foo-1.0-x86_64.AppImage",
		"Bar.appimage",   // case-insensitive match
		"NotAnAppImage",  // skipped
		"keep.me.txt",    // skipped
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("fake"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := scanAppImageDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 records (Foo + Bar); got %d: %+v", len(got), got)
	}

	names := map[string]bool{}
	for _, r := range got {
		names[r.Name] = true
		if r.Source != audit.SourceAppImage {
			t.Errorf("source: %q", r.Source)
		}
		if r.SizeBytes <= 0 {
			t.Errorf("size should be set: %+v", r)
		}
		if r.InstalledAt == nil {
			t.Errorf("install time should be set: %+v", r)
		}
		if !strings.HasPrefix(r.ReinstallCmd, "(redownload") {
			t.Errorf("reinstall hint: %q", r.ReinstallCmd)
		}
	}
	if !names["Foo-1.0-x86_64"] {
		t.Error("missing Foo record")
	}
}

func TestScanAppImageDirHandlesMissing(t *testing.T) {
	got, err := scanAppImageDir(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Errorf("missing dir should not error: %v", err)
	}
	if got != nil {
		t.Errorf("missing dir should return nil records; got %v", got)
	}
}
