package unversioned

import (
	"encoding/json"
	"testing"
)

func TestRuntimeSpecWindowsAddress(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "defaults to containerd pipe when unset",
			input: `{"name":"containerd","address":"unix:///run/containerd/containerd.sock"}`,
			want:  DefaultWindowsRuntimeAddress,
		},
		{
			name:  "defaults when only name is provided",
			input: `{"name":"containerd"}`,
			want:  DefaultWindowsRuntimeAddress,
		},
		{
			name:  "custom named pipe is preserved",
			input: `{"name":"containerd","windowsAddress":"npipe://./pipe/custom-containerd"}`,
			want:  "npipe://./pipe/custom-containerd",
		},
		{
			name:  "tcp endpoint is accepted",
			input: `{"name":"containerd","windowsAddress":"tcp://127.0.0.1:3459"}`,
			want:  "tcp://127.0.0.1:3459",
		},
		{
			name:    "invalid scheme is rejected",
			input:   `{"name":"containerd","windowsAddress":"unix:///run/containerd/containerd.sock"}`,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rs RuntimeSpec
			err := json.Unmarshal([]byte(tc.input), &rs)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none (WindowsAddress=%q)", rs.WindowsAddress)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rs.WindowsAddress != tc.want {
				t.Errorf("WindowsAddress = %q, want %q", rs.WindowsAddress, tc.want)
			}
		})
	}
}
