package version

import "runtime/debug"

// Build-time variables injected via -ldflags during release compilation.
var (
	// Version holds the current semver release or development tag.
	Version = "0.1.0-dev"

	// Commit holds the git commit SHA.
	Commit = "none"

	// Date holds the RFC3339 build timestamp.
	Date = "unknown"
)

// Info represents structured build metadata.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// Get returns the populated build information structure, falling back to Go module
// build info (e.g. when installed via `go install ...@vX.Y.Z` without ldflags).
func Get() Info {
	v := Version
	if v == "" || v == "0.1.0-dev" {
		if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			v = bi.Main.Version
		}
	}
	return Info{
		Version: v,
		Commit:  Commit,
		Date:    Date,
	}
}
