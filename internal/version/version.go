// Package version provides build-time version information.
package version

// These variables are set at build time using ldflags:
//
//	go build -ldflags "-X github.com/javanstorm/vmterminal/internal/version.Version=1.0.0 \
//	                   -X github.com/javanstorm/vmterminal/internal/version.Commit=$(git rev-parse HEAD) \
//	                   -X github.com/javanstorm/vmterminal/internal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
var (
	// Version is the semantic version of the application.
	Version = "dev"

	// Commit is the git commit SHA at build time.
	Commit = "unknown"

	// BuildDate is the date when the binary was built.
	BuildDate = "unknown"
)
