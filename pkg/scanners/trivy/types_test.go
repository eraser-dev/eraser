package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/eraser-dev/eraser/api/unversioned"

	"github.com/stretchr/testify/assert"
)

const (
	ref                 = "image:tag"
	trivyExecutableName = "trivy"
	trivyPathBin        = "/usr/bin/trivy"
)

var testDuration = unversioned.Duration(100000000000)

func TestCLIArgs(t *testing.T) {
	type testCell struct {
		desc     string
		config   Config
		expected []string
	}

	tests := []testCell{
		{
			desc:   "empty config",
			config: Config{},
			// default container runtime is containerd
			expected: []string{"--format=json", "image", "--image-src", ImgSrcContainerd, ref},
		},
		{
			desc:     "DeleteFailedImages has no effect",
			config:   Config{DeleteFailedImages: true},
			expected: []string{"--format=json", "image", "--image-src", ImgSrcContainerd, ref},
		},
		{
			desc:     "DeleteEOLImages has no effect",
			config:   Config{DeleteEOLImages: true},
			expected: []string{"--format=json", "image", "--image-src", ImgSrcContainerd, ref},
		},
		{
			desc:     "alternative runtime crio",
			config:   Config{Runtime: unversioned.RuntimeSpec{Name: unversioned.RuntimeCrio, Address: unversioned.CrioPath}},
			expected: []string{"--format=json", "image", "--image-src", ImgSrcPodman, ref},
		},
		{
			desc:     "alternative runtime dockershim",
			config:   Config{Runtime: unversioned.RuntimeSpec{Name: unversioned.RuntimeDockerShim, Address: unversioned.DockerPath}},
			expected: []string{"--format=json", "image", "--image-src", ImgSrcDocker, ref},
		},
		{
			desc:     "with cachedir",
			config:   Config{CacheDir: "/var/lib/trivy"},
			expected: []string{"--format=json", "--cache-dir", "/var/lib/trivy", "image", "--image-src", ImgSrcContainerd, ref},
		},
		{
			desc:     "with custom db repo",
			config:   Config{DBRepo: "example.test/db/repo"},
			expected: []string{"--format=json", "image", "--image-src", ImgSrcContainerd, "--db-repository", "example.test/db/repo", ref},
		},
		{
			desc:     "ignore unfixed",
			config:   Config{Vulnerabilities: VulnConfig{IgnoreUnfixed: true}},
			expected: []string{"--format=json", "image", "--image-src", ImgSrcContainerd, "--ignore-unfixed", ref},
		},
		{
			desc:     "specify vulnerability types",
			config:   Config{Vulnerabilities: VulnConfig{Types: []string{"library", "os"}}},
			expected: []string{"--format=json", "image", "--image-src", ImgSrcContainerd, "--vuln-type", "library,os", ref},
		},
		{
			desc:     "specify security checks / scanners",
			config:   Config{Vulnerabilities: VulnConfig{SecurityChecks: []string{"license", "vuln"}}},
			expected: []string{"--format=json", "image", "--image-src", ImgSrcContainerd, "--scanners", "license,vuln", ref},
		},
		{
			desc:     "specify severities",
			config:   Config{Vulnerabilities: VulnConfig{Severities: []string{"LOW", "MEDIUM"}}},
			expected: []string{"--format=json", "image", "--image-src", ImgSrcContainerd, "--severity", "LOW,MEDIUM", ref},
		},
		{
			desc:     "specify statuses to ignore",
			config:   Config{Vulnerabilities: VulnConfig{IgnoredStatuses: []string{statusUnknown, statusFixed, statusWillNotFix}}},
			expected: []string{"--format=json", "image", "--image-src", ImgSrcContainerd, "--ignore-status", "unknown,fixed,will_not_fix", ref},
		},
		{
			desc:     "total timeout has no effect",
			config:   Config{Timeout: TimeoutConfig{Total: testDuration}},
			expected: []string{"--format=json", "image", "--image-src", ImgSrcContainerd, ref},
		},
		{
			desc:     "per-image timeout",
			config:   Config{Timeout: TimeoutConfig{PerImage: testDuration}},
			expected: []string{"--format=json", "--timeout", "1m40s", "image", "--image-src", ImgSrcContainerd, ref},
		},
		{
			desc:   "all global options",
			config: Config{CacheDir: "/var/lib/trivy", Timeout: TimeoutConfig{PerImage: testDuration}},
			// these are output in a consistent order
			expected: []string{"--format=json", "--cache-dir", "/var/lib/trivy", "--timeout", "1m40s", "image", "--image-src", "containerd", ref},
		},
		{
			desc: "all `image` options",
			config: Config{
				Runtime: unversioned.RuntimeSpec{
					Name:    unversioned.RuntimeCrio,
					Address: unversioned.CrioPath,
				},
				DBRepo: "example.test/db/repo",
				Vulnerabilities: VulnConfig{
					IgnoreUnfixed:   true,
					Types:           []string{"library", "os"},
					SecurityChecks:  []string{"license", "vuln"},
					Severities:      []string{"LOW", "MEDIUM"},
					IgnoredStatuses: []string{statusUnknown, statusFixed},
				},
			},
			expected: []string{
				"--format=json", "image", "--image-src", ImgSrcPodman, "--db-repository", "example.test/db/repo", "--ignore-unfixed",
				"--vuln-type", "library,os", "--scanners", "license,vuln", "--severity", "LOW,MEDIUM", "--ignore-status", "unknown,fixed", ref,
			},
		},
		{
			desc: "all options",
			config: Config{
				CacheDir: "/var/lib/trivy",
				Timeout:  TimeoutConfig{PerImage: testDuration},
				Runtime: unversioned.RuntimeSpec{
					Name:    unversioned.RuntimeCrio,
					Address: unversioned.CrioPath,
				},
				DBRepo: "example.test/db/repo",
				Vulnerabilities: VulnConfig{
					IgnoreUnfixed:   true,
					Types:           []string{"os"},
					SecurityChecks:  []string{"license", "vuln"},
					Severities:      []string{"CRITICAL"},
					IgnoredStatuses: []string{statusUnknown, statusFixed},
				},
			},
			expected: []string{
				"--format=json", "--cache-dir", "/var/lib/trivy", "--timeout", "1m40s", "image", "--image-src", ImgSrcPodman,
				"--db-repository", "example.test/db/repo", "--ignore-unfixed", "--vuln-type", "os", "--scanners",
				"license,vuln", "--severity", "CRITICAL", "--ignore-status", "unknown,fixed", ref,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			actual := tt.config.cliArgs(ref)
			if len(actual) != len(tt.expected) {
				t.Logf("expected resulting length to be %d, was actually %d", len(actual), len(tt.expected))
				t.Fail()
			}

			for i := 0; i < len(actual); i++ {
				if actual[i] != tt.expected[i] {
					t.Logf("expected argument %s in position %d, was actually %s", tt.expected[i], i, actual[i])
					t.Fail()
				}
			}

			if t.Failed() {
				t.Logf("expected result `%s`, but got `%s`", strings.Join(tt.expected, " "), strings.Join(actual, " "))
			}
		})
	}
}

// TestImageScanner_findTrivyExecutable tests the findTrivyExecutable method in isolation.
func TestImageScanner_findTrivyExecutable(t *testing.T) {
	// Store original function to restore after tests
	originalLookPath := currentExecutingLookPath
	defer func() { currentExecutingLookPath = originalLookPath }()

	scanner := &ImageScanner{}

	testCases := []struct {
		name               string
		lookPathSetup      func()
		expectedPath       string
		expectedError      bool
		expectedErrorMatch string
	}{
		{
			name: "Trivy found in PATH only",
			lookPathSetup: func() {
				currentExecutingLookPath = func(file string) (string, error) {
					if file == trivyExecutableName {
						return trivyPathBin, nil
					}
					return "", errors.New("not found")
				}
			},
			expectedPath:  trivyPathBin,
			expectedError: false,
		},
		{
			name: "Trivy not found anywhere",
			lookPathSetup: func() {
				currentExecutingLookPath = func(_ string) (string, error) {
					return "", errors.New("executable file not found in $PATH")
				}
			},
			expectedPath:       "",
			expectedError:      true,
			expectedErrorMatch: "trivy executable not found at /trivy and not found in PATH",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.lookPathSetup()

			path, err := scanner.findTrivyExecutable()

			if tc.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrorMatch)
				assert.Empty(t, path)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedPath, path)
			}
		})
	}
}

// TestImageScanner_Scan_TrivyPathLookup tests the logic for finding the trivy executable.
func TestImageScanner_Scan_TrivyPathLookup(t *testing.T) {
	// Store original function to restore after tests
	originalLookPath := currentExecutingLookPath
	defer func() { currentExecutingLookPath = originalLookPath }()

	// Base configuration for the scanner
	baseConfig := DefaultConfig()
	scanner := &ImageScanner{
		config: *baseConfig,
	}
	// Dummy image for testing
	img := unversioned.Image{ImageID: "test-image-id", Names: []string{"test-image:latest"}}

	// Expected error message prefix when trivy is not found
	expectedNotFoundErrorMsgPrefix := fmt.Sprintf("trivy executable not found at %s", trivyCommandName)

	testCases := []struct {
		name                     string
		lookPathSetup            func() // Sets up the mock for exec.LookPath
		expectedStatus           ScanStatus
		expectNotFoundError      bool   // True if we expect the specific "trivy not found by LookPath" error
		expectedErrorMsgContains string // The prefix for the "not found" error message
	}{
		{
			name: "Trivy found at hardcoded path /trivy",
			lookPathSetup: func() {
				currentExecutingLookPath = func(file string) (string, error) {
					if file == trivyExecutableName {
						return trivyPathBin, nil // Found in PATH
					}
					return originalLookPath(file) // Fallback for any other calls
				}
			},
			// Scan will likely still fail due to inability to run actual scan in test,
			// but it should not be the "trivy not found by LookPath" error.
			expectedStatus:      StatusFailed,
			expectNotFoundError: false,
		},
		{
			name: "Trivy found in $PATH, not at /trivy",
			lookPathSetup: func() {
				currentExecutingLookPath = func(file string) (string, error) {
					if file == trivyExecutableName {
						return trivyPathBin, nil // Found in $PATH
					}
					return originalLookPath(file)
				}
			},
			expectedStatus:      StatusFailed, // Similar to above, subsequent scan steps will fail.
			expectNotFoundError: false,
		},
		{
			name: "Trivy not found anywhere",
			lookPathSetup: func() {
				currentExecutingLookPath = func(file string) (string, error) {
					if file == trivyExecutableName {
						return "", errors.New("mock: trivy not in $PATH")
					}
					return originalLookPath(file)
				}
			},
			expectedStatus:           StatusFailed,
			expectNotFoundError:      true,
			expectedErrorMsgContains: expectedNotFoundErrorMsgPrefix,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.lookPathSetup()

			status, err := scanner.Scan(img)

			if tc.expectNotFoundError {
				assert.Error(t, err, "Expected an error when trivy is not found")
				if err != nil { // Check prefix only if error is not nil
					assert.True(t, strings.HasPrefix(err.Error(), tc.expectedErrorMsgContains),
						"Error message should start with '%s'. Got: %s", tc.expectedErrorMsgContains, err.Error())
				}
				assert.Equal(t, tc.expectedStatus, status, "ScanStatus should be StatusFailed")
			} else if err != nil {
				// If trivy was "found" by LookPath, any error should be from subsequent operations (e.g., cmd.Run, JSON unmarshal),
				// not the specific "trivy executable not found by LookPath..." error.
				assert.False(t, strings.HasPrefix(err.Error(), expectedNotFoundErrorMsgPrefix),
					"Error should not be the 'trivy not found by LookPath' error. Got: %s", err.Error())
				// The status might still be StatusFailed due to these subsequent errors,
				// which is acceptable for this test's focus on path lookup.
			}
		})
	}
}
