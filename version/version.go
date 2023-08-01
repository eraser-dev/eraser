package version

import (
	"fmt"
	"runtime"
)

var (
	// BuildVersion is version set on build.
	BuildVersion string
	// DefaultRepo is the default repo for images.
	DefaultRepo = "ghcr.io/eraser-dev"
	// buildTime is the date for the binary build.
	buildTime string
	// vcsCommit is the commit hash for the binary build.
	vcsCommit string
)

// GetUserAgent returns a user agent of the format eraser/<component>/<version> (<goos>/<goarch>) <commit>/<timestamp>.
func GetUserAgent(component string) string {
	return fmt.Sprintf("eraser/%s/%s (%s/%s) %s/%s", component, BuildVersion, runtime.GOOS, runtime.GOARCH, vcsCommit, buildTime)
}
