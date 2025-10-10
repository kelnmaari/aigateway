// Package version provides version information for the Ollama-OpenAI Proxy
package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version (set via ldflags)
	Version = "dev"

	// GitCommit is the git commit hash (set via ldflags)
	GitCommit = "unknown"

	// BuildDate is the build date (set via ldflags)
	BuildDate = "unknown"

	// GoVersion is the Go version used to build the binary
	GoVersion = runtime.Version()
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
}

// GetInfo returns version information
func GetInfo() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
	}
}

// String returns version as a formatted string
func (i Info) String() string {
	return fmt.Sprintf(
		"Version: %s, Commit: %s, Built: %s, Go: %s",
		i.Version,
		i.GitCommit,
		i.BuildDate,
		i.GoVersion,
	)
}

// Short returns a short version string
func Short() string {
	if GitCommit != "unknown" && len(GitCommit) > 7 {
		return fmt.Sprintf("%s-%s", Version, GitCommit[:7])
	}
	return Version
}
