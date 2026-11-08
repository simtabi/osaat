package restore

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
)

// seedDir writes a representative scan output to a temp dir so the
// round-trip tests have realistic content.
func seedDir(t *testing.T) (dir string, contents map[string][]byte) {
	t.Helper()
	dir = t.TempDir()
	contents = map[string][]byte{
		"report.json":  []byte(`{"schema":"osaat.report/v1","records":[]}`),
		"report.md":    []byte("# Report\n\nNothing yet.\n"),
		"secrets.json": []byte(`{"schema":"osaat.secrets/v1","categories":[]}`),
		"Brewfile":     []byte(`cask "1password"` + "\n"),
		"mas-apps.txt": []byte("497799835  # Xcode\n"),
		"RESTORE.md":   []byte("# Restore checklist\n"),
		"SHA256SUMS":   []byte("aaa  report.json\nbbb  Brewfile\n"),
		"notes.txt":    []byte("a stray file the user dropped here\n"),
	}
	for name, body := range contents {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(filepath.Join(dir, "secrets.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, contents
}

func TestWriteAndDecryptArchiveRoundTrip(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("GenerateX25519Identity: %v", err)
	}

	srcDir, original := seedDir(t)

	var buf bytes.Buffer
	if err := WriteArchive(&buf, ArchiveOptions{
		SourceDir:  srcDir,
		Recipients: []age.Recipient{identity.Recipient()},
	}); err != nil {
		t.Fatalf("WriteArchive: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("archive should not be empty")
	}

	// The encrypted file should NOT contain plaintext markers.
	for _, marker := range []string{"osaat.report/v1", "1password", "Xcode"} {
		if bytes.Contains(buf.Bytes(), []byte(marker)) {
			t.Errorf("plaintext leaked: %q found in encrypted bytes", marker)
		}
	}

	outDir := t.TempDir()
	written, err := DecryptArchive(&buf, outDir, []age.Identity{identity})
	if err != nil {
		t.Fatalf("DecryptArchive: %v", err)
	}

	// Each known-set file in the seed should have round-tripped.
	wantedSet := map[string]bool{}
	for name := range original {
		for _, known := range DefaultArchiveFiles {
			if name == known {
				wantedSet[name] = true
				break
			}
		}
	}

	got := map[string]bool{}
	for _, p := range written {
		got[filepath.Base(p)] = true
	}
	for name := range wantedSet {
		if !got[name] {
			t.Errorf("missing %q in decrypted output (got %v)", name, written)
		}
	}
	// `notes.txt` is not in DefaultArchiveFiles → should NOT appear.
	if got["notes.txt"] {
		t.Errorf("stray notes.txt should have been skipped (set IncludeExtras=true to keep it)")
	}

	// Byte equality on a sample file.
	out, err := os.ReadFile(filepath.Join(outDir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, original["report.json"]) {
		t.Errorf("report.json content drifted across round-trip")
	}
}

func TestWriteArchiveIncludeExtras(t *testing.T) {
	identity, _ := age.GenerateX25519Identity()
	srcDir, _ := seedDir(t)

	var buf bytes.Buffer
	if err := WriteArchive(&buf, ArchiveOptions{
		SourceDir:     srcDir,
		Recipients:    []age.Recipient{identity.Recipient()},
		IncludeExtras: true,
	}); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	written, _ := DecryptArchive(&buf, outDir, []age.Identity{identity})

	var foundExtras bool
	for _, p := range written {
		if filepath.Base(p) == "notes.txt" {
			foundExtras = true
			break
		}
	}
	if !foundExtras {
		t.Errorf("IncludeExtras=true should keep notes.txt; got %v", written)
	}
}

func TestWriteArchiveRequiresRecipient(t *testing.T) {
	srcDir, _ := seedDir(t)
	var buf bytes.Buffer
	err := WriteArchive(&buf, ArchiveOptions{SourceDir: srcDir})
	if err == nil {
		t.Fatal("expected error for empty recipients")
	}
	if !strings.Contains(err.Error(), "recipient") {
		t.Errorf("error should mention recipient: %v", err)
	}
}

func TestDecryptArchiveRequiresIdentity(t *testing.T) {
	_, err := DecryptArchive(bytes.NewReader([]byte("garbage")), t.TempDir(), nil)
	if err == nil {
		t.Fatal("expected error for empty identities")
	}
}

func TestParseRecipientsRejectsBadKey(t *testing.T) {
	_, err := ParseRecipients([]string{"not-a-key"})
	if err == nil {
		t.Fatal("expected error for bad recipient")
	}
}

func TestSafeNameRejectsTraversal(t *testing.T) {
	for _, bad := range []string{"../etc/passwd", "/absolute/path", "..", ""} {
		if err := safeName(bad); err == nil {
			t.Errorf("safeName(%q) should error", bad)
		}
	}
	for _, ok := range []string{"report.json", "subdir/file.txt"} {
		if err := safeName(ok); err != nil {
			t.Errorf("safeName(%q) returned %v", ok, err)
		}
	}
}
