package restore

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"filippo.io/age"
)

// DefaultArchiveFiles is the set of artifact names that `osaat backup`
// pulls out of a scan output directory and into the encrypted bundle.
// Files outside this list are skipped unless includeExtras is true —
// the goal is a known-shaped bundle that won't pick up stray notes
// the user left in the same directory.
var DefaultArchiveFiles = []string{
	"report.json",
	"report.md",
	"report.csv",
	"report.html",
	"report.txt",
	"report.pdf",
	"secrets.json",
	"secrets.json.age",
	"Brewfile",
	"mas-apps.txt",
	"apt-packages.txt",
	"dnf-packages.txt",
	"pacman-packages.txt",
	"RESTORE.md",
	"SHA256SUMS",
	"osaat-metadata.json",
}

// ArchiveOptions configures a WriteArchive call.
type ArchiveOptions struct {
	// SourceDir is the scan output directory to bundle.
	SourceDir string
	// Recipients are the age public keys the archive should be
	// encrypted to.
	Recipients []age.Recipient
	// IncludeExtras includes every regular file under SourceDir,
	// not just the DefaultArchiveFiles set.
	IncludeExtras bool
}

// WriteArchive streams a tar of the eligible files in opts.SourceDir,
// encrypts the stream with age to opts.Recipients, and writes the
// ciphertext to w. The encrypted layer wraps tar — i.e. the file is
// in age envelope format, and decrypting reveals a tar stream.
func WriteArchive(w io.Writer, opts ArchiveOptions) error {
	if len(opts.Recipients) == 0 {
		return errors.New("WriteArchive requires at least one age recipient")
	}
	files, err := selectFiles(opts.SourceDir, opts.IncludeExtras)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no archivable files found in %s", opts.SourceDir)
	}

	enc, err := age.Encrypt(w, opts.Recipients...)
	if err != nil {
		return fmt.Errorf("age encrypt: %w", err)
	}
	tw := tar.NewWriter(enc)

	for _, f := range files {
		if err := addFileToTar(tw, opts.SourceDir, f); err != nil {
			_ = tw.Close()
			_ = enc.Close()
			return err
		}
	}

	if err := tw.Close(); err != nil {
		_ = enc.Close()
		return fmt.Errorf("close tar: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("close age writer: %w", err)
	}
	return nil
}

// DecryptArchive reverses WriteArchive: reads an age ciphertext from
// r, decrypts with the supplied identities, and extracts the tar
// contents into outDir. Returns the list of paths written.
func DecryptArchive(r io.Reader, outDir string, identities []age.Identity) ([]string, error) {
	if len(identities) == 0 {
		return nil, errors.New("DecryptArchive requires at least one age identity")
	}
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return nil, fmt.Errorf("create %s: %w", outDir, err)
	}

	dec, err := age.Decrypt(r, identities...)
	if err != nil {
		return nil, fmt.Errorf("age decrypt: %w", err)
	}

	tr := tar.NewReader(dec)
	var written []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return written, fmt.Errorf("tar read: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if err := safeName(hdr.Name); err != nil {
			return written, err
		}
		outPath := filepath.Join(outDir, hdr.Name)
		if err := writeFromTar(tr, outPath, os.FileMode(hdr.Mode).Perm()); err != nil {
			return written, err
		}
		written = append(written, outPath)
	}
	return written, nil
}

// ParseRecipients parses one or more age recipient strings ("age1...").
// Empty entries are skipped; an entry that doesn't parse causes the
// whole call to fail so the user finds typos before the encrypted file
// is written.
func ParseRecipients(specs []string) ([]age.Recipient, error) {
	out := make([]age.Recipient, 0, len(specs))
	for _, s := range specs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		r, err := age.ParseX25519Recipient(s)
		if err != nil {
			return nil, fmt.Errorf("parse age recipient %q: %w", s, err)
		}
		out = append(out, r)
	}
	if len(out) == 0 {
		return nil, errors.New("no age recipients provided")
	}
	return out, nil
}

// LoadIdentitiesFromFile reads an age identity file (the format
// produced by `age-keygen -o ~/.age/key.txt`) and returns every
// identity it contains.
func LoadIdentitiesFromFile(path string) ([]age.Identity, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	ids, err := age.ParseIdentities(f)
	if err != nil {
		return nil, fmt.Errorf("parse identities from %s: %w", path, err)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no age identities found in %s", path)
	}
	return ids, nil
}

func selectFiles(srcDir string, includeExtras bool) ([]string, error) {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", srcDir, err)
	}
	want := make(map[string]bool, len(DefaultArchiveFiles))
	if !includeExtras {
		for _, name := range DefaultArchiveFiles {
			want[name] = true
		}
	}
	var picked []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if includeExtras || want[name] {
			picked = append(picked, name)
		}
	}
	sort.Strings(picked)
	return picked, nil
}

func addFileToTar(tw *tar.Writer, srcDir, name string) error {
	full := filepath.Join(srcDir, name)
	info, err := os.Stat(full)
	if err != nil {
		return fmt.Errorf("stat %s: %w", full, err)
	}
	hdr := &tar.Header{
		Name:    name,
		Size:    info.Size(),
		Mode:    int64(info.Mode().Perm()),
		ModTime: info.ModTime().UTC().Truncate(time.Second),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("tar header %s: %w", name, err)
	}
	f, err := os.Open(full)
	if err != nil {
		return fmt.Errorf("open %s: %w", full, err)
	}
	defer f.Close()
	if _, err := io.Copy(tw, f); err != nil {
		return fmt.Errorf("tar copy %s: %w", name, err)
	}
	return nil
}

func writeFromTar(tr *tar.Reader, outPath string, mode os.FileMode) error {
	if mode == 0 {
		mode = 0o644
	}
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("create %s: %w", outPath, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, tr); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	return nil
}

// safeName rejects tar entry names that would escape outDir via
// absolute paths or `..` traversal. The DefaultArchiveFiles set
// is well-known but a hostile or corrupt archive could ship anything.
func safeName(name string) error {
	if name == "" {
		return errors.New("empty name in tar entry")
	}
	if filepath.IsAbs(name) {
		return fmt.Errorf("absolute path in tar entry: %q", name)
	}
	cleaned := filepath.Clean(name)
	if strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || cleaned == ".." {
		return fmt.Errorf("path traversal in tar entry: %q", name)
	}
	return nil
}
