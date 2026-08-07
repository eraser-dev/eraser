package util

import (
	"testing"

	"k8s.io/klog/v2"
	"oras.land/oras-go/v2/registry"
)

func TestParseRepoTag(t *testing.T) {
	cases := []struct {
		input     string
		expected  RepoTag
		expectErr bool
	}{
		{
			input: "ghcr.io/repo/one/two:three",
			expected: RepoTag{
				Repo: "ghcr.io/repo/one/two",
				Tag:  "three",
			},
			expectErr: false,
		},
		{
			input: "ghcr.io/one:two",
			expected: RepoTag{
				Repo: "ghcr.io/one",
				Tag:  "two",
			},
			expectErr: false,
		},
		{
			input: "eraser:e2e-test",
			expected: RepoTag{
				Repo: "eraser",
				Tag:  "e2e-test",
			},
			expectErr: false,
		},
		{
			input: "eraser@sha256:4dca0fd5f424a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc",
			expected: RepoTag{
				Repo: "eraser",
				Tag:  "sha256:4dca0fd5f424a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc",
			},
			expectErr: false,
		},
		{
			input: "eraser:sha256:4dca0fd5f424a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc",
			expected: RepoTag{
				Repo: "eraser",
				Tag:  "sha256:4dca0fd5f424a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc",
			},
			expectErr: false,
		},
		{
			input: "docker.io/nginx:sha256:4dca0fd5f424a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc",
			expected: RepoTag{
				Repo: "docker.io/nginx",
				Tag:  "sha256:4dca0fd5f424a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc",
			},
			expectErr: false,
		},
		{
			input: "localhost:5000/repo:sha256:4dca0fd5f424a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc",
			expected: RepoTag{
				Repo: "localhost:5000/repo",
				Tag:  "sha256:4dca0fd5f424a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc",
			},
			expectErr: false,
		},
		{
			input: "eraser@sha256:4dca0fd5f4:4a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc",
			expected: RepoTag{
				Repo: "",
				Tag:  "",
			},
			expectErr: true,
		},
		{
			input: "docker.io/nginx@sha256:4dca0fd5f424a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc",
			expected: RepoTag{
				Repo: "docker.io/nginx",
				Tag:  "sha256:4dca0fd5f424a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc",
			},
			expectErr: false,
		},
		{
			input: "docker.io/library/nginx@sha256:4dca0fd5f424a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc",
			expected: RepoTag{
				Repo: "docker.io/library/nginx",
				Tag:  "sha256:4dca0fd5f424a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc",
			},
			expectErr: false,
		},
		{
			input: "docker.io/nginx@sha256:4dca0fd5f4",
			expected: RepoTag{
				Repo: "",
				Tag:  "",
			},
			expectErr: true,
		},
		{
			input: "docker.io/nginx@sha256:gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg",
			expected: RepoTag{
				Repo: "",
				Tag:  "",
			},
			expectErr: true,
		},
		{
			input: "docker.io/library/nginx@sha123:4dca0fd5f424a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc",
			expected: RepoTag{
				Repo: "",
				Tag:  "",
			},
			expectErr: true,
		},
		{
			input: "",
			expected: RepoTag{
				Repo: "",
				Tag:  "",
			},
			expectErr: false,
		},
		{
			input: ":",
			expected: RepoTag{
				Repo: "",
				Tag:  "",
			},
			expectErr: true,
		},
		{
			input: "/",
			expected: RepoTag{
				Repo: "",
				Tag:  "",
			},
			expectErr: true,
		},
	}

	for _, c := range cases {
		result, err := parseRepoTag(c.input)
		if err != nil {
			if c.expectErr {
				continue
			}

			klog.Errorf("error from parsing function: %#v\ninput: %s\nexpected: %#v\ngot:      %#v", err, c.input, c.expected, result)
			t.FailNow()
		}

		if c.expectErr {
			klog.Errorf("expected error parsing reference `%s`, but did not receive one", c.input)
			t.Fail()
		}

		if result.Repo != c.expected.Repo || result.Tag != c.expected.Tag {
			klog.Errorf("wrong result\ninput: %s\nexpected: %#v\ngot:      %#v", c.input, c.expected, result)
			t.Fail()
		}
	}
}

func TestParseRegistryReferenceLegacyDigest(t *testing.T) {
	const digest = "sha256:4dca0fd5f424a31b03ab807cbae77eb32bf2d089eed1cee154b3afed458de0dc"

	cases := []struct {
		name       string
		input      string
		registry   string
		repository string
	}{
		{
			name:       "registry",
			input:      "docker.io/nginx:" + digest,
			registry:   "docker.io",
			repository: "nginx",
		},
		{
			name:       "registry with port",
			input:      "localhost:5000/repo:" + digest,
			registry:   "localhost:5000",
			repository: "repo",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := registry.ParseReference(tc.input); err == nil {
				t.Fatalf("oras-go v2 unexpectedly accepted legacy digest reference %q; compatibility fallback may be removable", tc.input)
			}

			got, err := parseRegistryReference(tc.input)
			if err != nil {
				t.Fatalf("parseRegistryReference(%q) returned error: %v", tc.input, err)
			}
			if got.Registry != tc.registry || got.Repository != tc.repository || got.Reference != digest {
				t.Fatalf(
					"parseRegistryReference(%q) = %#v, want registry=%q repository=%q reference=%q",
					tc.input,
					got,
					tc.registry,
					tc.repository,
					digest,
				)
			}
		})
	}
}
