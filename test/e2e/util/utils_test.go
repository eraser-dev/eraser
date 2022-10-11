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
			expectErr: false,
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

			klog.Errorf("error from parsing function: %#v\nexpected: %#v\ngot:      %#v", err, c.expected, result)
			t.FailNow()
		}

		if result.Repo != c.expected.Repo || result.Tag != c.expected.Tag {
			klog.Errorf("wrong result\nexpected: %#v\ngot:      %#v", c.expected, result)
			t.Fail()
		}
	}
}
