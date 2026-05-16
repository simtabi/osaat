// Package reporters renders []audit.AppRecord into one of the
// user-facing output formats. Each reporter is a pure function over
// the record set — it never imports collectors directly.
package reporters

import (
	"io"

	"github.com/simtabi/osaat/internal/audit"
)

// Reporter writes a record set in a given format.
type Reporter interface {
	// Format is the short token used on the --format CLI flag and as
	// the file extension in default output paths.
	Format() string
	// Write encodes records to w. The implementation owns sorting and
	// envelope shape.
	Write(records []audit.AppRecord, w io.Writer) error
}
