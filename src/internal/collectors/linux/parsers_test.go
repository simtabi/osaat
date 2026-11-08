package linux

import (
	"context"
	"testing"

	"github.com/simtabi/osaat/internal/audit"
)

func TestParseDpkgOutput(t *testing.T) {
	// Realistic dpkg-query output. Last record has half-installed
	// status — should be skipped.
	out := []byte("" +
		"bash\t5.2.21-2ubuntu4\tUbuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>\t1816\tinstalled\tshells\thttps://www.gnu.org/software/bash/\n" +
		"curl\t8.5.0-2ubuntu10.5\tCurl Maintainers <curl@curl.se>\t506\tinstalled\tweb\thttps://curl.se\n" +
		"libfoo\t1.2.3-1\tDebian Multimedia <debian-multimedia@lists.debian.org>\t52\tconfig-files\tlibs\t\n" +
		"")
	got := parseDpkgOutput(out)
	if len(got) != 2 {
		t.Fatalf("want 2 records (half-installed skipped); got %d: %+v", len(got), got)
	}
	if got[0].Name != "bash" || got[0].Source != audit.SourceDpkg {
		t.Errorf("record 0: %+v", got[0])
	}
	if got[0].ReinstallCmd != "apt install bash" {
		t.Errorf("reinstall_cmd: %q", got[0].ReinstallCmd)
	}
	if got[0].Author != "Ubuntu Developers" {
		t.Errorf("maintainer name parsing: %q", got[0].Author)
	}
	if got[0].VendorURL != "https://www.gnu.org/software/bash/" {
		t.Errorf("homepage: %q", got[0].VendorURL)
	}
	if got[0].SizeBytes != 1816*1024 {
		t.Errorf("size: got %d, want %d (dpkg reports KB)", got[0].SizeBytes, 1816*1024)
	}
}

func TestParseDpkgSkipsMalformedLines(t *testing.T) {
	out := []byte("" +
		"\n" +
		"only one field\n" +
		"\t\t\t\tinstalled\n" + // empty name
		"good\t1.0\tMaintainer <m@x>\t100\tinstalled\tcat\thttp://x\n" +
		"")
	got := parseDpkgOutput(out)
	if len(got) != 1 || got[0].Name != "good" {
		t.Fatalf("expected 1 good record; got %+v", got)
	}
}

func TestParseRpmOutput(t *testing.T) {
	out := []byte("" +
		"bash\t5.2.15-5.fc40\tFedora Project\t8253440\t1717000000\thttps://www.gnu.org/software/bash/\n" +
		"curl\t8.6.0-9.fc40\tFedora Project\t423936\t1717000123\t(none)\n" +
		"libnone\t1.0\t(none)\t1024\t0\t(none)\n" +
		"")
	got := parseRpmOutput(out, "dnf")
	if len(got) != 3 {
		t.Fatalf("want 3 records; got %d", len(got))
	}
	if got[0].Source != audit.SourceRpm || got[0].ReinstallCmd != "dnf install bash" {
		t.Errorf("bash: %+v", got[0])
	}
	if got[0].SizeBytes != 8253440 {
		t.Errorf("bash size: got %d", got[0].SizeBytes)
	}
	if got[0].InstalledAt == nil || got[0].InstalledAt.Unix() != 1717000000 {
		t.Errorf("bash installed_at: %v", got[0].InstalledAt)
	}
	if got[1].VendorURL != "" {
		t.Errorf("curl url should be empty when source said (none); got %q", got[1].VendorURL)
	}
	if got[2].Author != "" {
		t.Errorf("(none) vendor should be empty; got %q", got[2].Author)
	}
}

func TestParseRpmAlternateFrontend(t *testing.T) {
	out := []byte("foo\t1.0\tSUSE\t1024\t0\t\n")
	got := parseRpmOutput(out, "zypper")
	if len(got) != 1 || got[0].ReinstallCmd != "zypper install foo" {
		t.Errorf("zypper frontend: %+v", got)
	}
}

func TestParsePacmanOutput(t *testing.T) {
	out := []byte("" +
		"bash 5.2.026-3\n" +
		"curl 8.7.1-2\n" +
		"linux 6.8.7.arch1-1\n" +
		"\n" +
		"  trailing-space   1.0  \n" +
		"")
	got := parsePacmanOutput(out)
	if len(got) != 4 {
		t.Fatalf("want 4 records; got %d: %+v", len(got), got)
	}
	if got[2].Name != "linux" || got[2].Version != "6.8.7.arch1-1" {
		t.Errorf("linux record: %+v", got[2])
	}
	if got[0].ReinstallCmd != "pacman -S bash" {
		t.Errorf("reinstall_cmd: %q", got[0].ReinstallCmd)
	}
	if got[0].Source != audit.SourcePacman {
		t.Errorf("source: %q", got[0].Source)
	}
}

func TestExtractEmailFromMaintainer(t *testing.T) {
	cases := []struct {
		in, wantName, wantEmail string
	}{
		{"Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>", "Ubuntu Developers", "ubuntu-devel-discuss@lists.ubuntu.com"},
		{"Curl Maintainers <curl@curl.se>", "Curl Maintainers", "curl@curl.se"},
		{"No Email Here", "No Email Here", ""},
		{"", "", ""},
	}
	for _, tc := range cases {
		name, email := extractEmailFromMaintainer(tc.in)
		if name != tc.wantName {
			t.Errorf("name from %q: got %q, want %q", tc.in, name, tc.wantName)
		}
		if email != tc.wantEmail {
			t.Errorf("email from %q: got %q, want %q", tc.in, email, tc.wantEmail)
		}
	}
}

func TestEpochToTime(t *testing.T) {
	if epochToTime("") != nil {
		t.Error("empty epoch should return nil")
	}
	if epochToTime("0") != nil {
		t.Error("epoch 0 should return nil")
	}
	if epochToTime("notanumber") != nil {
		t.Error("invalid epoch should return nil")
	}
	got := epochToTime("1717000000")
	if got == nil {
		t.Fatal("valid epoch should return non-nil")
	}
	if got.Unix() != 1717000000 {
		t.Errorf("epoch round-trip: got %d", got.Unix())
	}
}

// TestCollectIntegration is a smoke test that exercises the
// orchestrator with a fake RunCmd. It runs on every OS.
func TestCollectIntegration(t *testing.T) {
	fake := func(name string) []byte {
		switch name {
		case "dpkg-query":
			return []byte("bash\t5.2.21\tDeb <d@x>\t100\tinstalled\tshells\thttps://gnu.org\n")
		case "rpm":
			return []byte("kernel\t6.8\tRed Hat\t1000\t1700000000\thttps://kernel.org\n")
		case "pacman":
			return []byte("vim 9.1.0-1\n")
		}
		return nil
	}
	c := New(WithRunCmd(func(_ context.Context, name string, _ ...string) ([]byte, error) {
		return fake(name), nil
	}))
	// We can't call Collect() here because of the GOOS guard; test the
	// individual functions instead (they don't gate on GOOS).
	if got, err := c.collectDpkg(contextTODO()); err != nil || len(got) != 1 || got[0].Name != "bash" {
		t.Errorf("dpkg path: %v / %+v", err, got)
	}
	if got, err := c.collectRpm(contextTODO()); err != nil || len(got) != 1 || got[0].Name != "kernel" {
		t.Errorf("rpm path: %v / %+v", err, got)
	}
	if got, err := c.collectPacman(contextTODO()); err != nil || len(got) != 1 || got[0].Name != "vim" {
		t.Errorf("pacman path: %v / %+v", err, got)
	}
}

// contextTODO is a tiny indirection so we don't need to import
// context just for the test fixture above.
func contextTODO() context.Context { return context.TODO() }
