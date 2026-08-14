//go:build !windows

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
			endpoint: fmt.Sprintf("unix://%s", testUnixPath),
			addr:     testUnixPath,
			err:      nil,
		},
		{
			endpoint: "localhost:8080",
			addr:     "",
			err:      ErrProtocolNotSupported,
		},
		{
			endpoint: "tcp://localhost:8080",
			addr:     "",
			err:      ErrOnlySupportUnixSocket,
		},
	}

	for _, tc := range testCases {
		a, _, e := getAddressAndDialer(tc.endpoint)
		if a != tc.addr || !errors.Is(e, tc.err) {
			t.Errorf("getAddressAndDialer(%q) = (%q, %v), want (%q, %v)", tc.endpoint, a, e, tc.addr, tc.err)
		}
	}
}

func TestDefaultCRIPathIsUnixSocket(t *testing.T) {
	if CRIPath != "/run/cri/cri.sock" {
		t.Errorf("CRIPath = %q, want /run/cri/cri.sock", CRIPath)
	}
}
