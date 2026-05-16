// Package profiles persists wizard answer sets as TOML files under
// ~/.config/osaat/profiles/<name>.toml, with mode 600.
package profiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/simtabi/osaat/internal/paths"
)

// SchemaProfileV1 is the value of the "schema" field in a profile.
const SchemaProfileV1 = "osaat.profile/v1"

// Profile is the persisted shape of wizard answers.
type Profile struct {
	Schema   string   `toml:"schema"`
	OS       []string `toml:"os,omitempty"`
	Formats  []string `toml:"formats,omitempty"`
	Out      string   `toml:"out,omitempty"`
	License  License  `toml:"license,omitempty"`
	Insights Insights `toml:"insights,omitempty"`
	Restore  Restore  `toml:"restore,omitempty"`
}

// License holds license-extraction settings.
type License struct {
	Mode         string `toml:"mode,omitempty"`
	AgeRecipient string `toml:"age_recipient,omitempty"`
}

// Insights holds the optional column toggles.
type Insights struct {
	Forgotten       bool `toml:"forgotten,omitempty"`
	ForgottenMonths int  `toml:"forgotten_months,omitempty"`
	AppleSilicon    bool `toml:"apple_silicon,omitempty"`
}

// Restore controls whether the restoration manifest is emitted.
type Restore struct {
	Enabled bool `toml:"enabled,omitempty"`
}

// Save serializes p as TOML to ~/.config/osaat/profiles/<name>.toml.
// The directory is created with mode 700; the file with mode 600.
func Save(name string, p Profile) (string, error) {
	if err := validateName(name); err != nil {
		return "", err
	}
	dir, err := paths.ProfilesDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create profile dir: %w", err)
	}
	p.Schema = SchemaProfileV1
	data, err := toml.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("marshal profile: %w", err)
	}
	path := filepath.Join(dir, name+".toml")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

// Load reads a profile by name. Schema mismatches surface as errors
// (so future readers don't quietly accept incompatible files).
func Load(name string) (Profile, error) {
	if err := validateName(name); err != nil {
		return Profile{}, err
	}
	dir, err := paths.ProfilesDir()
	if err != nil {
		return Profile{}, err
	}
	path := filepath.Join(dir, name+".toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, fmt.Errorf("read profile %s: %w", name, err)
	}
	var p Profile
	if err := toml.Unmarshal(data, &p); err != nil {
		return Profile{}, fmt.Errorf("parse profile %s: %w", name, err)
	}
	if p.Schema != "" && p.Schema != SchemaProfileV1 {
		return p, fmt.Errorf("profile %s has unknown schema %q (expected %q)", name, p.Schema, SchemaProfileV1)
	}
	return p, nil
}

// List returns the names of every profile on disk, sorted.
func List() ([]string, error) {
	dir, err := paths.ProfilesDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, ".toml") {
			names = append(names, strings.TrimSuffix(n, ".toml"))
		}
	}
	sort.Strings(names)
	return names, nil
}

// Exists reports whether a profile with this name is on disk.
func Exists(name string) (bool, error) {
	if err := validateName(name); err != nil {
		return false, err
	}
	dir, err := paths.ProfilesDir()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(filepath.Join(dir, name+".toml"))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// validateName rejects names that could escape the profiles directory
// (path traversal) or contain shell-troublesome characters.
func validateName(name string) error {
	if name == "" {
		return errors.New("profile name is empty")
	}
	if strings.ContainsAny(name, "/\\:") {
		return fmt.Errorf("profile name %q must not contain / \\ or :", name)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("profile name %q is not allowed", name)
	}
	return nil
}
