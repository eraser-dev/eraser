package main

import (
	"strings"
	"testing"
)

const ref = "image:tag"

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
			expected: []string{"--format=json", "image", "--image-src", "containerd", "image:tag"},
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
