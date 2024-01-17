package main

import (
	"strings"
	"testing"

	"github.com/eraser-dev/eraser/api/unversioned"
)

const ref = "image:tag"

var testDuration = unversioned.Duration(100000000000)

func init() {
}

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
