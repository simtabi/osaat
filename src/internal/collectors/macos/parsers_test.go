package macos

import (
	"testing"

	"github.com/simtabi/osaat/internal/audit"
)

func TestParseMasList(t *testing.T) {
	out := []byte(`497799835 Xcode    (16.0)
1295203466 Microsoft Remote Desktop    (10.9.10)
12345 NotAnApp (this is not a version
909901780 Stickies    (1.0)
`)
	entries := parseMasList(out)
	if got, want := len(entries), 3; got != want {
		t.Fatalf("got %d entries, want %d: %#v", got, want, entries)
	}
	if entries[0].ID != "497799835" || entries[0].Name != "Xcode" || entries[0].Version != "16.0" {
		t.Errorf("entry 0 wrong: %+v", entries[0])
	}
	if entries[1].Name != "Microsoft Remote Desktop" {
		t.Errorf("entry 1 name wrong: %q", entries[1].Name)
	}
}

func TestParseBrewCaskList(t *testing.T) {
	out := []byte(`1password 8.10.39
firefox 130.0
adobe-creative-cloud
zoom 5.17.10
`)
	entries := parseBrewCaskList(out)
	if got, want := len(entries), 4; got != want {
		t.Fatalf("got %d entries, want %d", got, want)
	}
	if entries[2].Cask != "adobe-creative-cloud" || entries[2].Version != "" {
		t.Errorf("entry 2 wrong: %+v", entries[2])
	}
	if entries[0].Version != "8.10.39" {
		t.Errorf("entry 0 version wrong: %q", entries[0].Version)
	}
}

func TestParsePkgList(t *testing.T) {
	out := []byte("com.apple.pkg.Core\ncom.example.foo\n   \ncom.simtabi.bar\n")
	got := parsePkgList(out)
	want := []string{"com.apple.pkg.Core", "com.example.foo", "com.simtabi.bar"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d]=%q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseMdlsWhereFroms(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "two-url tuple",
			in: `kMDItemWhereFroms = (
    "https://example.com/foo.dmg",
    "https://example.com/"
)`,
			want: "https://example.com/foo.dmg",
		},
		{
			name: "null",
			in:   `kMDItemWhereFroms = (null)`,
			want: "",
		},
		{
			name: "non-url",
			in: `kMDItemWhereFroms = (
    "Safari"
)`,
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseMdlsWhereFroms([]byte(tc.in))
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseQuarantineAgent(t *testing.T) {
	cases := map[string]string{
		"0001;6537e2a3;Safari;UUID-here": "Safari",
		"0083;abcd;Chrome;F-2":           "Chrome",
		"shortvalue":                     "",
		"":                               "",
	}
	for in, want := range cases {
		if got := parseQuarantineAgent([]byte(in)); got != want {
			t.Errorf("parseQuarantineAgent(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseCodesignSigned(t *testing.T) {
	out := `Executable=/Applications/Safari.app/Contents/MacOS/Safari
Identifier=com.apple.Safari
Format=app bundle with Mach-O thin (arm64)
CodeDirectory v=20500 size=98765 flags=0x12000(library-validation,runtime)
Authority=Software Signing
Authority=Apple Code Signing Certification Authority
Authority=Apple Root CA
TeamIdentifier=APPLECORP
Signature=signed
Info.plist entries=33
`
	status, team, authority := parseCodesign(out)
	if status != audit.SigningSigned {
		t.Errorf("status: got %q, want %q", status, audit.SigningSigned)
	}
	if team != "APPLECORP" {
		t.Errorf("team: got %q, want APPLECORP", team)
	}
	if authority != "Software Signing" {
		t.Errorf("authority: got %q, want Software Signing", authority)
	}
}

func TestParseCodesignAdHoc(t *testing.T) {
	out := `Executable=/Applications/Foo.app/Contents/MacOS/Foo
Identifier=com.example.foo
Format=app bundle with Mach-O thin (arm64)
TeamIdentifier=not set
Signature=adhoc
`
	status, team, _ := parseCodesign(out)
	if status != audit.SigningAdHoc {
		t.Errorf("status: got %q, want %q", status, audit.SigningAdHoc)
	}
	if team != "" {
		t.Errorf("team: got %q, want empty (codesign reports 'not set')", team)
	}
}

func TestParseCodesignUnsigned(t *testing.T) {
	out := "/Applications/Foo.app: code object is not signed at all"
	status, _, _ := parseCodesign(out)
	if status != audit.SigningUnsigned {
		t.Errorf("status: got %q, want %q", status, audit.SigningUnsigned)
	}
}

func TestMapObtainedFrom(t *testing.T) {
	cases := []struct {
		obtained string
		current  audit.Source
		want     audit.Source
	}{
		{"mac_app_store", audit.SourceUnknown, audit.SourceAppStore},
		{"apple", audit.SourceUnknown, audit.SourceSystem},
		{"identified_developer", audit.SourceUnknown, audit.SourceDMG},
		{"identified_developer", audit.SourceBrewCask, audit.SourceBrewCask}, // don't override more-specific source
		{"unknown", audit.SourceBrewCask, audit.SourceBrewCask},
		{"", audit.SourceUnknown, audit.SourceUnknown},
	}
	for _, tc := range cases {
		got := mapObtainedFrom(tc.obtained, tc.current)
		if got != tc.want {
			t.Errorf("mapObtainedFrom(%q, %q) = %q, want %q", tc.obtained, tc.current, got, tc.want)
		}
	}
}
