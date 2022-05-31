package version

import (
	"fmt"
	"runtime"
	"testing"
	"strings"
)

func TestGetUserAgent(t *testing.T) {
	buildTime = "Now"
	buildVersion = "version"
	vcsCommit = "hash"

	expected := fmt.Sprintf("eraser/manager/%s (%s/%s) %s/%s", buildVersion, runtime.GOOS, runtime.GOARCH, vcsCommit, buildTime)
	actual := GetUserAgent("manager")
	if !strings.EqualFold(expected, actual) {
		t.Fatalf("expected: %s, got: %s", expected, actual)
	}
}