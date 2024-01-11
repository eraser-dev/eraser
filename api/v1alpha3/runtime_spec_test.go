package v1alpha3

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestConvertRuntimeToRuntimeSpec(t *testing.T) {
	type testCase struct {
		input     Runtime
		expected  RuntimeSpec
		shouldErr bool
	}

	tests := map[string]testCase{
		"Containerd": {
			input:     RuntimeContainerd,
			expected:  RuntimeSpec{Name: RuntimeContainerd, Address: fmt.Sprintf("unix://%s", ContainerdPath)},
			shouldErr: false,
		},
		"DockerShim": {
			input:     RuntimeDockerShim,
			expected:  RuntimeSpec{Name: RuntimeDockerShim, Address: fmt.Sprintf("unix://%s", DockerPath)},
			shouldErr: false,
		},
		"Crio": {
			input:     RuntimeCrio,
			expected:  RuntimeSpec{Name: RuntimeCrio, Address: fmt.Sprintf("unix://%s", CrioPath)},
			shouldErr: false,
		},
		"InvalidRuntime": {
			input:     "invalid",
			expected:  RuntimeSpec{},
			shouldErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := ConvertRuntimeToRuntimeSpec(test.input)

			if test.shouldErr && err == nil {
				t.Errorf("Expected an error but got nil")
			}

			if !test.shouldErr && err != nil {
				t.Errorf("Error: %v", err)
			}

			if result != test.expected {
				t.Errorf("Unexpected result. Expected %v, but got %v", test.expected, result)
			}
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	type testCase struct {
		input     []byte
		expected  RuntimeSpec
		shouldErr bool
	}

	tests := map[string]testCase{
		"ValidContainerd": {
			input:     []byte(`{"name": "containerd", "address": "unix:///run/containerd/containerd.sock"}`),
			expected:  RuntimeSpec{Name: RuntimeContainerd, Address: fmt.Sprintf("unix://%s", ContainerdPath)},
			shouldErr: false,
		},
		"ValidDockerShim": {
			input:     []byte(`{"name": "dockershim", "address": "unix:///run/dockershim.sock"}`),
			expected:  RuntimeSpec{Name: RuntimeDockerShim, Address: fmt.Sprintf("unix://%s", DockerPath)},
			shouldErr: false,
		},
		"ValidCrio": {
			input:     []byte(`{"name": "crio", "address": "unix:///run/crio/crio.sock"}`),
			expected:  RuntimeSpec{Name: RuntimeCrio, Address: fmt.Sprintf("unix://%s", CrioPath)},
			shouldErr: false,
		},
		"InvalidName": {
			input:     []byte(`{"name": "invalid", "address": "unix:///invalid"}`),
			expected:  RuntimeSpec{},
			shouldErr: true,
		},
		"InvalidAddressScheme": {
			input:     []byte(`{"name": "containerd", "address": "http://invalid"}`),
			expected:  RuntimeSpec{},
			shouldErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var rs RuntimeSpec
			err := json.Unmarshal(test.input, &rs)

			if test.shouldErr && err == nil {
				t.Error("Expected an error but got nil")
			}

			if !test.shouldErr && err != nil {
				t.Errorf("Error: %v", err)
			}

			if rs != test.expected {
				t.Errorf("Unexpected result. Expected %v, but got %v", test.expected, rs)
			}
		})
	}
}
