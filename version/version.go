package version

import (
	"fmt"
	"runtime"
)

var (
	// buildVersion is version set on build.
	buildVersion string
	// buildTime is the date for the binary build.
	buildTime string
	// vcsCommit is the commit hash for the binary build.
	vcsCommit string
)

var (
	// DefaultRepo is the default repo for images
	DefaultRepo = "ghcr.io/azure"
	// DefaultTag is the default tag for images
	DefaultTag = "latest"
)

// GetUserAgent returns a user agent of the format eraser/<component>/<version> (<goos>/<goarch>) <commit>/<timestamp>.
func GetUserAgent(component string) string {
	return fmt.Sprintf("eraser/%s/%s (%s/%s) %s/%s", component, buildVersion, runtime.GOOS, runtime.GOARCH, vcsCommit, buildTime)
}
