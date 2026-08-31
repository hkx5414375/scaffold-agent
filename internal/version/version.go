// Package version exposes build information for the Scaffold Agent binary.
package version

var (
	// Version is replaced by release builds through -ldflags.
	Version = "0.0.0-dev"
	// Commit is the source revision used for the build.
	Commit = "unknown"
	// BuildDate is the UTC build timestamp.
	BuildDate = "unknown"
)

// Info is the stable JSON representation of build metadata.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

// Current returns the current build metadata.
func Current() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
	}
}
