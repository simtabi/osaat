package secrets

import (
	"encoding/json"
	"fmt"
	"io"

	"filippo.io/age"
)

// WriteJSON writes f to w as indented JSON. Used for the unencrypted
// path — the resulting file should still be mode 600 on disk to
// avoid casual disclosure.
func WriteJSON(f *File, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(f)
}

// WriteEncrypted encrypts the JSON-encoded File to the given age
// recipient(s) and writes the ciphertext to w. Recipients are
// supplied as age public keys (the `age1...` form).
//
// Use a recipients-file form by passing one recipient per line —
// each recipient is parsed independently.
func WriteEncrypted(f *File, recipients []string, w io.Writer) error {
	if len(recipients) == 0 {
		return fmt.Errorf("no age recipients provided")
	}

	parsed, err := ParseRecipients(recipients)
	if err != nil {
		return err
	}

	enc, err := age.Encrypt(w, parsed...)
	if err != nil {
		return fmt.Errorf("age encrypt: %w", err)
	}
	if err := WriteJSON(f, enc); err != nil {
		_ = enc.Close()
		return err
	}
	return enc.Close()
}

// ParseRecipients accepts a list of recipient strings (typically
// `age1...` X25519 public keys, optionally with whitespace) and
// returns the corresponding age.Recipient values.
func ParseRecipients(recipients []string) ([]age.Recipient, error) {
	out := make([]age.Recipient, 0, len(recipients))
	for _, r := range recipients {
		if r == "" {
			continue
		}
		rec, err := age.ParseX25519Recipient(r)
		if err != nil {
			return nil, fmt.Errorf("parse age recipient %q: %w", r, err)
		}
		out = append(out, rec)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no usable age recipients")
	}
	return out, nil
}
