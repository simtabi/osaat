package macos

import (
	"os"
	"path/filepath"
	"testing"
)

const infoPlistXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>TestApp</string>
    <key>CFBundleDisplayName</key>
    <string>Test App (Display)</string>
    <key>CFBundleIdentifier</key>
    <string>com.example.testapp</string>
    <key>CFBundleShortVersionString</key>
    <string>1.2.3</string>
    <key>CFBundleVersion</key>
    <string>1234</string>
    <key>CFBundleExecutable</key>
    <string>TestApp</string>
</dict>
</plist>
`

// Some real apps ship a dict where a string is expected. The decoder
// must not crash on those — asString returns "" silently.
const infoPlistWithDictField = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <dict>
        <key>en</key>
        <string>WrongType</string>
    </dict>
    <key>CFBundleIdentifier</key>
    <string>com.example.unusual</string>
    <key>CFBundleShortVersionString</key>
    <string>9.9.9</string>
</dict>
</plist>
`

func TestReadInfoPlistRegular(t *testing.T) {
	dir := t.TempDir()
	plistPath := filepath.Join(dir, "Info.plist")
	if err := os.WriteFile(plistPath, []byte(infoPlistXML), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := readInfoPlist(plistPath)
	if err != nil {
		t.Fatalf("readInfoPlist: %v", err)
	}
	if info.CFBundleName != "TestApp" {
		t.Errorf("CFBundleName: got %q", info.CFBundleName)
	}
	if info.CFBundleDisplayName != "Test App (Display)" {
		t.Errorf("CFBundleDisplayName: got %q", info.CFBundleDisplayName)
	}
	if info.CFBundleIdentifier != "com.example.testapp" {
		t.Errorf("CFBundleIdentifier: got %q", info.CFBundleIdentifier)
	}
	if info.CFBundleShortVersionString != "1.2.3" {
		t.Errorf("CFBundleShortVersionString: got %q", info.CFBundleShortVersionString)
	}
}

func TestReadInfoPlistToleratesWrongFieldTypes(t *testing.T) {
	dir := t.TempDir()
	plistPath := filepath.Join(dir, "Info.plist")
	if err := os.WriteFile(plistPath, []byte(infoPlistWithDictField), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := readInfoPlist(plistPath)
	if err != nil {
		t.Fatalf("readInfoPlist with dict-typed field should not fail: %v", err)
	}
	if info.CFBundleName != "" {
		t.Errorf("CFBundleName should be empty when typed as dict; got %q", info.CFBundleName)
	}
	if info.CFBundleIdentifier != "com.example.unusual" {
		t.Errorf("CFBundleIdentifier: got %q", info.CFBundleIdentifier)
	}
}

func TestAsString(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{"hello", "hello"},
		{[]byte("bytes"), "bytes"},
		{nil, ""},
		{map[string]any{"k": "v"}, ""},
		{[]any{"a", "b"}, ""},
		{42, ""},
	}
	for _, tc := range cases {
		got := asString(tc.in)
		if got != tc.want {
			t.Errorf("asString(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
