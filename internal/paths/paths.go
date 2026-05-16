// Package paths centralizes OS-aware default paths used across osaat.
// The defaults intentionally match the user-facing convention from the
// project plan: Documents folder for generated artifacts on every OS,
// ~/.config/osaat/ for config + logs + profiles regardless of OS.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// DefaultDocumentsDir returns the OS-standard user Documents directory
// joined with "osaat" — this is the default base for generated audit
// outputs.
//
// macOS / Windows: $HOME/Documents/osaat (or %USERPROFILE%/Documents/osaat).
// Linux / BSD: $XDG_DOCUMENTS_DIR/osaat, else $HOME/Documents/osaat.
func DefaultDocumentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	var docs string
	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd", "netbsd":
		if x := strings.TrimSpace(os.Getenv("XDG_DOCUMENTS_DIR")); x != "" {
			docs = x
		} else {
			docs = filepath.Join(home, "Documents")
		}
	default:
		docs = filepath.Join(home, "Documents")
	}
	return filepath.Join(docs, "osaat"), nil
}

// DefaultOutputDir returns Documents/osaat/YYYY-MM-DD — distinct per
// day so subsequent runs don't overwrite an earlier audit.
func DefaultOutputDir() (string, error) {
	docs, err := DefaultDocumentsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(docs, time.Now().Format("2006-01-02")), nil
}

// ConfigDir returns $HOME/.config/osaat — the same path on every OS,
// per the project decision to keep config layout uniform.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "osaat"), nil
}

// ProfilesDir returns the subdirectory under ConfigDir that holds
// named profile TOML files.
func ProfilesDir() (string, error) {
	cfg, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "profiles"), nil
}

// LogsDir returns the subdirectory under ConfigDir for daily log
// files (osaat-YYYY-MM-DD.log).
func LogsDir() (string, error) {
	cfg, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "logs"), nil
}

// TidyPath substitutes $HOME with "~" for display. Use in user-facing
// strings; never use the result as an actual path.
func TidyPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

// ScrubHome replaces every occurrence of $HOME in s with "~". Used by
// the log scrubber to keep usernames out of files on disk.
func ScrubHome(s string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return s
	}
	return strings.ReplaceAll(s, home, "~")
}
