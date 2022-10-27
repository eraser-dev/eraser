package util

import (
	"testing"

	"k8s.io/klog/v2"
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
