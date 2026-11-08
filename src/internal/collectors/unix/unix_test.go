package unix

import (
	"testing"

	"github.com/simtabi/osaat/internal/audit"
)

func TestParseFreeBSDPkgOutput(t *testing.T) {
	out := []byte("" +
		"nginx\t1.24.0_5,2\twww@FreeBSD.org\t1717000000\twww/nginx\thttps://nginx.org/\n" +
		"postgresql15-server\t15.4_1\tpgsql@FreeBSD.org\t1718000000\tdatabases/postgresql15-server\thttps://www.postgresql.org\n" +
		"bare\t1.0\t\t\t\t\n" +
		"")
	got := parseFreeBSDPkgOutput(out)
	if len(got) != 3 {
		t.Fatalf("want 3 records; got %d", len(got))
	}
	if got[0].Name != "nginx" || got[0].Version != "1.24.0_5,2" {
		t.Errorf("nginx record: %+v", got[0])
	}
	if got[0].Author != "www@FreeBSD.org" {
		t.Errorf("maintainer: %q", got[0].Author)
	}
	if got[0].VendorURL != "https://nginx.org/" {
		t.Errorf("url: %q", got[0].VendorURL)
	}
	if got[0].ReinstallCmd != "pkg install -y nginx" {
		t.Errorf("reinstall: %q", got[0].ReinstallCmd)
	}
	if got[0].Source != audit.SourceBSDPkg {
		t.Errorf("source: %q", got[0].Source)
	}
	if got[0].InstalledAt == nil || got[0].InstalledAt.Unix() != 1717000000 {
		t.Errorf("installed_at: %v", got[0].InstalledAt)
	}
	// "bare" had empty time / no url — should still produce a record
	// without crashing and without InstalledAt set.
	if got[2].InstalledAt != nil {
		t.Errorf("empty epoch should leave installed_at nil; got %v", got[2].InstalledAt)
	}
}

func TestParsePkgInfoOutput(t *testing.T) {
	// Captured `pkg_info` output from OpenBSD 7.x.
	out := []byte("" +
		"nginx-1.24.0nb2     High performance HTTP server\n" +
		"vim-9.0.1234       Improved version of the venerable vi editor\n" +
		"glib2-2.78.4       Some core library used by GNOME\n" +
		"bad-line-with-no-version  description here\n" +
		"")
	got := parsePkgInfoOutput(out)
	if len(got) != 3 {
		t.Fatalf("want 3 valid records (bad line skipped); got %d", len(got))
	}
	if got[0].Name != "nginx" || got[0].Version != "1.24.0nb2" {
		t.Errorf("nginx record: %+v", got[0])
	}
	if got[1].ReinstallCmd != "pkg_add vim" {
		t.Errorf("vim reinstall: %q", got[1].ReinstallCmd)
	}
	if got[2].Name != "glib2" {
		t.Errorf("glib2 name: %q", got[2].Name)
	}
}

func TestEpochSecondsToTimePtr(t *testing.T) {
	if epochSecondsToTimePtr("") != nil {
		t.Error("empty should return nil")
	}
	if epochSecondsToTimePtr("0") != nil {
		t.Error("0 should return nil")
	}
	if epochSecondsToTimePtr("notnum") != nil {
		t.Error("non-numeric should return nil")
	}
	got := epochSecondsToTimePtr("1718000000")
	if got == nil || got.Unix() != 1718000000 {
		t.Errorf("valid epoch: %v", got)
	}
}
