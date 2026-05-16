// Package version exposes build-time identifying information for the
// osaat binary. The values are overridden at link time by goreleaser
// and the Makefile via -ldflags.
package version

// Version is the semantic version of the binary, or "0.0.0-dev" in
// untagged local builds.
var Version = "0.0.0-dev"

// Commit is the short git SHA the binary was built from, or "unknown".
var Commit = "unknown"

// Date is the build timestamp in RFC 3339 UTC, or "unknown".
var Date = "unknown"

// String returns a human-readable version line.
func String() string {
	return Version + " (commit " + Commit + ", built " + Date + ")"
}
