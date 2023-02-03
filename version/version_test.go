package version

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

func TestGetUserAgent(t *testing.T) {
	buildTime = "Now"
	BuildVersion = "version"
	vcsCommit = "hash"

	expected := fmt.Sprintf("eraser/manager/%s (%s/%s) %s/%s", BuildVersion, runtime.GOOS, runtime.GOARCH, vcsCommit, buildTime)
	actual := GetUserAgent("manager")
	if !strings.EqualFold(expected, actual) {
		t.Fatalf("expected: %s, got: %s", expected, actual)
	}
}
