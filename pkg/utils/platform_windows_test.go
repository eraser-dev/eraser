//go:build windows

package utils

import (
	"errors"
	"fmt"
	"testing"
)

func TestGetAddressAndDialer(t *testing.T) {
	testCases := []struct {
		endpoint string
		addr     string
		err      error
	}{
		{
			endpoint: CRIPath,
			addr:     `\\.\pipe\containerd-containerd`,
			err:      nil,
		},
		{
			// bare pipe name falls back to the npipe protocol
			endpoint: "./pipe/containerd-containerd",
			addr:     `\\.\pipe\containerd-containerd`,
			err:      nil,
		},
		{
			endpoint: fmt.Sprintf("unix://%s", testUnixPath),
			addr:     "",
			err:      ErrProtocolNotSupported,
		},
		{
			endpoint: "tcp://localhost:8080",
			addr:     "",
			err:      ErrProtocolNotSupported,
		},
	}

	for _, tc := range testCases {
		a, _, e := getAddressAndDialer(tc.endpoint)
		if a != tc.addr || !errors.Is(e, tc.err) {
			t.Errorf("getAddressAndDialer(%q) = (%q, %v), want (%q, %v)", tc.endpoint, a, e, tc.addr, tc.err)
		}
	}
}

func TestMkfifoUnsupported(t *testing.T) {
	if err := mkfifo("ignored", PipeMode); !errors.Is(err, ErrFifoUnsupported) {
		t.Errorf("mkfifo on windows = %v, want ErrFifoUnsupported", err)
	}
}
