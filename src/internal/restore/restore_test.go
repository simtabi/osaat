package restore

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simtabi/osaat/internal/audit"
)

func fixtureRecords() []audit.AppRecord {
	return []audit.AppRecord{
		{Name: "1Password", Source: audit.SourceBrewCask, ReinstallCmd: "brew install --cask 1password"},
		{Name: "Firefox", Source: audit.SourceBrewCask, ReinstallCmd: "brew install --cask firefox"},
		{Name: "Xcode", Source: audit.SourceAppStore, AppStoreID: "497799835"},
		{Name: "BlueHarvest", Source: audit.SourceAppStore /* no id — mas wasn't installed */},
		{Name: "Adobe Creative Cloud", Source: audit.SourceDMG, Author: "Adobe Inc.", VendorURL: "https://adobe.com", DownloadURL: "https://adobe.com/download.dmg"},
		{Name: "MyKernelExtension", Source: audit.SourcePkg, BundleID: "com.example.mke", Version: "1.2"},
		{Name: "Safari", Source: audit.SourceSystem},
	}
}

func TestWriteBrewfile(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteBrewfile(fixtureRecords(), &buf); err != nil {
		t.Fatalf("WriteBrewfile: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		`cask "1password"`,
		`cask "firefox"`,
		`mas "Xcode", id: 497799835`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Brewfile missing %q\n--- output ---\n%s", want, out)
		}
	}
	for _, unwanted := range []string{
		"Adobe Creative Cloud",
		"Safari",
		"MyKernelExtension",
		"BlueHarvest", // no mas ID, should not be in Brewfile
	} {
		if strings.Contains(out, unwanted) {
			t.Errorf("Brewfile should not contain %q (not auto-installable)\n%s", unwanted, out)
		}
	}
}

func TestWriteMasList(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteMasList(fixtureRecords(), &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "497799835") || !strings.Contains(out, "Xcode") {
		t.Errorf("mas-apps.txt missing Xcode/497799835\n%s", out)
	}
	if strings.Contains(out, "BlueHarvest") {
		t.Errorf("mas-apps.txt should not include apps without an ID\n%s", out)
	}
}

func TestWriteRestoreDoc(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteRestoreDoc(fixtureRecords(), &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"# Manual restore checklist",
		"## App Store (no mas ID",
		"BlueHarvest",
		"## Direct download",
		"Adobe Creative Cloud",
		"https://adobe.com",
		"## Installer packages",
		"MyKernelExtension",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("RESTORE.md missing %q", want)
		}
	}
	for _, unwanted := range []string{
		"1Password", // homebrew_cask, covered by Brewfile
		"Xcode",     // mas_apps covered
		"Safari",    // system, skipped
	} {
		if strings.Contains(out, unwanted) {
			t.Errorf("RESTORE.md should not contain %q (already covered)", unwanted)
		}
	}
}

func TestWriteAllIntegration(t *testing.T) {
	dir := t.TempDir()
	paths, err := WriteAll(fixtureRecords(), dir)
	if err != nil {
		t.Fatalf("WriteAll: %v", err)
	}
	want := []string{"Brewfile", "mas-apps.txt", "RESTORE.md"}
	if len(paths) != len(want) {
		t.Fatalf("paths: got %v, want %d files", paths, len(want))
	}
	for i, name := range want {
		expected := filepath.Join(dir, name)
		if paths[i] != expected {
			t.Errorf("path[%d]: got %q, want %q", i, paths[i], expected)
		}
		if info, err := os.Stat(expected); err != nil || info.Size() == 0 {
			t.Errorf("%s missing or empty: stat=%v err=%v", expected, info, err)
		}
	}
}
