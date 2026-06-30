// Package version exposes build metadata for velox.
package version

import "runtime"

// These values are overridable at build time via -ldflags
// (e.g. -X github.com/koller-nexus/velox/internal/version.Version=v1.2.3).
var (
	// Version is the semantic version of the build.
	Version = "0.1.0-dev"
	// Commit is the git commit the binary was built from.
	Commit = "none"
	// Date is the build date.
	Date = "unknown"
)

// String returns a human-readable version line. It performs no I/O and never
// touches the network, so it is safe to call for `--version` while offline.
func String() string {
	return "velox " + Version + " (commit " + Commit + ", built " + Date +
		", " + runtime.Version() + " " + runtime.GOOS + "/" + runtime.GOARCH + ")"
}
