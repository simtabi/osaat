package secrets

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
)

func sampleFile() *File {
	return &File{
		Schema:      SchemaSecretsV1,
		GeneratedAt: time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC),
		Host:        "test-host",
		LicenseMode: "best-effort",
		Categories: []Category{
			{
				Name: "Standalone (direct download)",
				Entries: []Entry{
					{
						AppName:    "TestApp",
						BundleID:   "com.example.testapp",
						LicenseKey: "ABCD-EFGH-IJKL-MNOP-QRST",
						Source:     "/Users/test/Library/Preferences/com.example.testapp.plist#LicenseKey",
						Confidence: ConfidenceHigh,
					},
				},
			},
		},
		ManualChecklist: []ChecklistItem{
			{AppName: "OtherApp", LookHere: "~/Library/Preferences/...", LookFor: "license email"},
		},
	}
}

func TestWriteJSONRoundTrip(t *testing.T) {
	f := sampleFile()
	var buf bytes.Buffer
	if err := WriteJSON(f, &buf); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var back File
	if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, buf.String())
	}
	if back.Schema != SchemaSecretsV1 {
		t.Errorf("schema: got %q", back.Schema)
	}
	if len(back.Categories) != 1 || len(back.Categories[0].Entries) != 1 {
		t.Fatalf("categories shape wrong: %+v", back.Categories)
	}
	if back.Categories[0].Entries[0].LicenseKey != "ABCD-EFGH-IJKL-MNOP-QRST" {
		t.Errorf("license key lost: %q", back.Categories[0].Entries[0].LicenseKey)
	}
}

func TestWriteEncryptedRoundTrip(t *testing.T) {
	// Generate an ephemeral identity for the test.
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("GenerateX25519Identity: %v", err)
	}
	recipient := identity.Recipient().String()

	f := sampleFile()
	var buf bytes.Buffer
	if err := WriteEncrypted(f, []string{recipient}, &buf); err != nil {
		t.Fatalf("WriteEncrypted: %v", err)
	}

	// Decrypt with the identity, verify we get the same File back.
	dec, err := age.Decrypt(&buf, identity)
	if err != nil {
		t.Fatalf("age.Decrypt: %v", err)
	}

	var back File
	if err := json.NewDecoder(dec).Decode(&back); err != nil {
		t.Fatalf("decode plaintext: %v", err)
	}
	if back.Categories[0].Entries[0].LicenseKey != "ABCD-EFGH-IJKL-MNOP-QRST" {
		t.Errorf("license key lost through encrypt round trip: %q", back.Categories[0].Entries[0].LicenseKey)
	}

	// Sanity: the encrypted output should NOT contain the plaintext key.
	if strings.Contains(buf.String(), "ABCD-EFGH-IJKL-MNOP-QRST") {
		t.Errorf("plaintext license key leaked into encrypted file")
	}
}

func TestWriteEncryptedRejectsEmptyRecipients(t *testing.T) {
	var buf bytes.Buffer
	err := WriteEncrypted(sampleFile(), nil, &buf)
	if err == nil {
		t.Fatal("expected error for empty recipients")
	}
	if !strings.Contains(err.Error(), "recipient") {
		t.Errorf("error should mention recipient: %v", err)
	}
}

func TestIsEmpty(t *testing.T) {
	if !(&File{}).IsEmpty() {
		t.Error("blank file should be empty")
	}
	if (&File{ManualChecklist: []ChecklistItem{{AppName: "X"}}}).IsEmpty() {
		t.Error("file with checklist should not be empty")
	}
	if (sampleFile()).IsEmpty() {
		t.Error("sample file should not be empty")
	}
}
