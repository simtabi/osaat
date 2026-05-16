package macos

import (
	"context"
	"testing"

	"github.com/simtabi/osaat/internal/audit"
)

const sysProfilerXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<array>
  <dict>
    <key>_dataType</key>
    <string>SPApplicationsDataType</string>
    <key>_items</key>
    <array>
      <dict>
        <key>_name</key>
        <string>StoreApp</string>
        <key>path</key>
        <string>/Applications/StoreApp.app</string>
        <key>obtained_from</key>
        <string>mac_app_store</string>
        <key>signed_by</key>
        <array>
          <string>Apple Mac OS Application Signing</string>
        </array>
        <key>version</key>
        <string>1.0.0</string>
      </dict>
      <dict>
        <key>_name</key>
        <string>DevApp</string>
        <key>path</key>
        <string>/Applications/DevApp.app</string>
        <key>obtained_from</key>
        <string>identified_developer</string>
        <key>signed_by</key>
        <array>
          <string>Developer ID Application: Acme Corp</string>
          <string>Developer ID Certification Authority</string>
        </array>
        <key>version</key>
        <string>2.3.4</string>
      </dict>
      <dict>
        <key>_name</key>
        <string>SysApp</string>
        <key>path</key>
        <string>/System/Applications/SysApp.app</string>
        <key>obtained_from</key>
        <string>apple</string>
        <key>signed_by</key>
        <array>
          <string>Software Signing</string>
        </array>
        <key>version</key>
        <string>14.5</string>
      </dict>
    </array>
  </dict>
</array>
</plist>
`

func TestEnrichFromSystemProfiler(t *testing.T) {
	mockRun := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte(sysProfilerXML), nil
	}
	c := New(WithRunCmd(mockRun))
	records := []audit.AppRecord{
		{Name: "StoreApp", Path: "/Applications/StoreApp.app", Source: audit.SourceUnknown},
		{Name: "DevApp", Path: "/Applications/DevApp.app", Source: audit.SourceUnknown},
		{Name: "SysApp", Path: "/System/Applications/SysApp.app", Source: audit.SourceUnknown},
		{Name: "Untouched", Path: "/Applications/Untouched.app", Source: audit.SourceUnknown},
	}

	if err := c.enrichFromSystemProfiler(context.Background(), records); err != nil {
		t.Fatalf("enrichFromSystemProfiler: %v", err)
	}

	if records[0].Source != audit.SourceAppStore {
		t.Errorf("StoreApp source: got %q, want %q", records[0].Source, audit.SourceAppStore)
	}
	if records[1].Source != audit.SourceDMG {
		t.Errorf("DevApp source: got %q, want %q (identified_developer maps to dmg_or_direct)", records[1].Source, audit.SourceDMG)
	}
	if records[1].Author != "Developer ID Application: Acme Corp" {
		t.Errorf("DevApp author: got %q", records[1].Author)
	}
	if records[2].Source != audit.SourceSystem {
		t.Errorf("SysApp source: got %q, want %q", records[2].Source, audit.SourceSystem)
	}
	if records[3].Source != audit.SourceUnknown {
		t.Errorf("Untouched source should stay Unknown when no system_profiler row matches; got %q", records[3].Source)
	}
}
