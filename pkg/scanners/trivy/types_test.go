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

// TestEnsureTrivyExecutable tests the ensureTrivyExecutable function in isolation.
func TestEnsureTrivyExecutable(t *testing.T) {
	// Save original PATH to restore after tests
	originalPath := os.Getenv("PATH")
	defer func() { os.Setenv("PATH", originalPath) }()

	testCases := []struct {
		name             string
		setupFunc        func(t *testing.T) (targetPath string, cleanup func())
		expectedError    bool
		expectedErrorMsg string
		validateFunc     func(t *testing.T, targetPath string)
	}{
		{
			name: "Trivy already exists at target path",
			setupFunc: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()
				targetPath := filepath.Join(tempDir, "trivy")

				// Create a trivy executable at the target path
				file, err := os.Create(targetPath)
				require.NoError(t, err)
				file.Close()
				err = os.Chmod(targetPath, 0o755)
				require.NoError(t, err)

				return targetPath, func() {}
			},
			expectedError: false,
			validateFunc: func(t *testing.T, targetPath string) {
				// Should not create a symlink, original file should still exist
				info, err := os.Lstat(targetPath)
				require.NoError(t, err)
				assert.Equal(t, os.FileMode(0o755), info.Mode().Perm(), "Original file should be preserved")

				// Verify it's not a symlink
				assert.Equal(t, 0, int(info.Mode()&os.ModeSymlink), "Should not be a symlink")
			},
		},
		{
			name: "Trivy found in PATH, symlink created successfully",
			setupFunc: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()
				pathDir := filepath.Join(tempDir, "bin")
				err := os.Mkdir(pathDir, 0o755)
				require.NoError(t, err)

				// Create a trivy executable in the PATH directory
				trivyInPath := filepath.Join(pathDir, "trivy")
				file, err := os.Create(trivyInPath)
				require.NoError(t, err)
				file.Close()
				err = os.Chmod(trivyInPath, 0o755)
				require.NoError(t, err)

				// Set PATH to include our temp bin directory
				os.Setenv("PATH", pathDir)

				// Target path where symlink should be created
				targetPath := filepath.Join(tempDir, "target_trivy")

				return targetPath, func() {}
			},
			expectedError: false,
			validateFunc: func(t *testing.T, targetPath string) {
				// Should create a symlink at target path
				info, err := os.Lstat(targetPath)
				require.NoError(t, err)

				// Verify it's a symlink
				assert.NotEqual(t, 0, int(info.Mode()&os.ModeSymlink), "Should be a symlink")

				// Verify symlink points to the correct location
				linkTarget, err := os.Readlink(targetPath)
				require.NoError(t, err)
				assert.Contains(t, linkTarget, "trivy", "Symlink should point to trivy executable")
			},
		},
		{
			name: "Trivy not found anywhere",
			setupFunc: func(t *testing.T) (string, func()) {
				tempDir := t.TempDir()
				emptyPathDir := filepath.Join(tempDir, "empty")
				err := os.Mkdir(emptyPathDir, 0o755)
				require.NoError(t, err)

				// Set PATH to empty directory without trivy
				os.Setenv("PATH", emptyPathDir)

				targetPath := filepath.Join(tempDir, "target_trivy")
				return targetPath, func() {}
			},
			expectedError:    true,
			expectedErrorMsg: "trivy executable not found",
			validateFunc: func(t *testing.T, targetPath string) {
				// Should not create any file or symlink
				_, err := os.Lstat(targetPath)
				assert.True(t, os.IsNotExist(err), "Target path should not exist when trivy is not found")
			},
		},
		{
			name: "Symlink creation fails due to permission",
			setupFunc: func(t *testing.T) (string, func()) {
				if os.Getuid() == 0 {
					t.Skip("Skipping permission test when running as root")
				}

				tempDir := t.TempDir()
				pathDir := filepath.Join(tempDir, "bin")
				err := os.Mkdir(pathDir, 0o755)
				require.NoError(t, err)

				// Create a trivy executable in PATH
				trivyInPath := filepath.Join(pathDir, "trivy")
				file, err := os.Create(trivyInPath)
				require.NoError(t, err)
				file.Close()
				err = os.Chmod(trivyInPath, 0o755)
				require.NoError(t, err)

				os.Setenv("PATH", pathDir)

				// Try to create symlink in root directory (should fail for non-root users)
				targetPath := "/trivy_test_symlink"

				return targetPath, func() {
					// Clean up any created symlink
					os.Remove(targetPath)
				}
			},
			expectedError:    true,
			expectedErrorMsg: "failed to create symlink",
			validateFunc: func(t *testing.T, targetPath string) {
				// Should not create symlink due to permission error
				_, err := os.Lstat(targetPath)
				assert.True(t, os.IsNotExist(err), "Target path should not exist when symlink creation fails")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetPath, cleanup := tc.setupFunc(t)
			defer cleanup()

			// Test the ensureTrivyExecutable function
			err := ensureTrivyExecutable(targetPath)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrorMsg)
			} else {
				assert.NoError(t, err)
			}

			if tc.validateFunc != nil {
				tc.validateFunc(t, targetPath)
			}
		})
	}
}

// TestInitScanner_TrivySymlinkCreation tests the symlink creation logic in initScanner.
func TestInitScanner_TrivySymlinkCreation(t *testing.T) {
	// This test verifies that initScanner properly calls ensureTrivyExecutable
	// and handles errors appropriately. Since ensureTrivyExecutable is tested
	// comprehensively above, this focuses on the integration aspect.

	// Save original PATH to restore after test
	originalPath := os.Getenv("PATH")
	defer func() { os.Setenv("PATH", originalPath) }()

	t.Run("initScanner fails when trivy not found", func(t *testing.T) {
		// Set PATH to empty directory
		tempDir := t.TempDir()
		os.Setenv("PATH", tempDir)

		config := DefaultConfig()
		scanner, err := initScanner(config)

		// Should fail because trivy is not found
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "trivy executable not found")
		assert.Nil(t, scanner)
	})
}
