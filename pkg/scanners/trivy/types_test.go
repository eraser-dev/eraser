package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eraser-dev/eraser/api/unversioned"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestFindTrivyExecutable tests the findTrivyExecutable function in isolation.
func TestFindTrivyExecutable(t *testing.T) {
	// Save original PATH to restore after tests
	originalPath := os.Getenv("PATH")
	defer func() { os.Setenv("PATH", originalPath) }()

	testCases := []struct {
		name               string
		setupFunc          func(t *testing.T) string // returns tempdir path
		expectedPath       string
		expectedError      bool
		expectedErrorMatch string
	}{
		{
			name: "Trivy found in PATH only",
			setupFunc: func(t *testing.T) string {
				tempDir := t.TempDir()
				trivyPath := filepath.Join(tempDir, "trivy")

				// Create executable file
				file, err := os.Create(trivyPath)
				require.NoError(t, err)
				file.Close()
				err = os.Chmod(trivyPath, 0o755)
				require.NoError(t, err)

				// Set PATH to include temp directory
				os.Setenv("PATH", tempDir)
				return trivyPath
			},
			expectedError: false,
		},
		{
			name: "Trivy not found anywhere",
			setupFunc: func(t *testing.T) string {
				// Set PATH to empty temp directory without trivy
				tempDir := t.TempDir()
				os.Setenv("PATH", tempDir)
				return ""
			},
			expectedPath:       "",
			expectedError:      true,
			expectedErrorMatch: "trivy executable not found at /trivy and not found in PATH",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expectedPath := tc.setupFunc(t)
			if expectedPath == "" {
				expectedPath = tc.expectedPath
			}

			path, err := findTrivyExecutable()

			if tc.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrorMatch)
				assert.Empty(t, path)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, expectedPath, path)
			}
		})
	}
}

// TestImageScanner_Scan_TrivyPathLookup tests the logic for using the trivy executable path.
func TestImageScanner_Scan_TrivyPathLookup(t *testing.T) {
	// Base configuration for the scanner
	baseConfig := DefaultConfig()
	// Dummy image for testing
	img := unversioned.Image{ImageID: "test-image-id", Names: []string{"test-image:latest"}}

	testCases := []struct {
		name           string
		trivyPath      string
		expectedStatus ScanStatus
	}{
		{
			name:           "Trivy path set to hardcoded path /trivy",
			trivyPath:      trivyCommandName,
			expectedStatus: StatusFailed, // Will fail during actual execution but not due to path issues
		},
		{
			name:           "Trivy path set to system PATH location",
			trivyPath:      trivyPathBin,
			expectedStatus: StatusFailed, // Will fail during actual execution but not due to path issues
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scanner := &ImageScanner{
				config:    *baseConfig,
				trivyPath: tc.trivyPath,
			}

			status, err := scanner.Scan(img)

			// The scan will likely fail due to inability to run actual scan in test,
			// but it should not be a "trivy not found" error since the path is already set
			assert.Equal(t, tc.expectedStatus, status, "ScanStatus should be StatusFailed")
			if err != nil {
				assert.NotContains(t, err.Error(), "trivy executable not found",
					"Error should not be about trivy not being found since path is pre-set")
			}
		})
	}
}
