package version

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

// Get returns the populated build information structure.
func Get() Info {
	return Info{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
	}
}
