package utils

import (
	"errors"
	"fmt"
	"net/url"
	"testing"
)

// testUnixPath is used instead of CRIPath so these cases stay valid on Windows,
// where CRIPath is a named pipe.
const testUnixPath = "/run/cri/cri.sock"

func TestParseEndpointWithFallBackProtocol(t *testing.T) {
	testCases := []struct {
		endpoint         string
		fallbackProtocol string
		protocol         string
		addr             string
		errCheck         func(t *testing.T, err error)
	}{
		{
			endpoint:         fmt.Sprintf("unix://%s", testUnixPath),
			fallbackProtocol: "unix",
			protocol:         "unix",
			addr:             testUnixPath,
			errCheck: func(t *testing.T, err error) {
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			endpoint:         "./pipe/containerd-containerd",
			fallbackProtocol: "npipe",
			protocol:         "npipe",
			addr:             `\\.\pipe\containerd-containerd`,
			errCheck: func(t *testing.T, err error) {
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			endpoint:         "192.168.123.132",
			fallbackProtocol: "unix",
			protocol:         "unix",
			addr:             "",
			errCheck: func(t *testing.T, err error) {
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			endpoint:         "tcp://localhost:8080",
			fallbackProtocol: "unix",
			protocol:         "tcp",
			addr:             "localhost:8080",
			errCheck: func(t *testing.T, err error) {
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			endpoint:         "  ",
			fallbackProtocol: "unix",
			protocol:         "",
			addr:             "",
			errCheck: func(t *testing.T, err error) {
				as := &url.Error{}
				if !errors.As(err, &as) {
					t.Error(err)
				}
			},
		},
	}

	for _, tc := range testCases {
		p, a, e := ParseEndpointWithFallbackProtocol(tc.endpoint, tc.fallbackProtocol)

		if p != tc.protocol || a != tc.addr {
			t.Errorf("Test fails")
		}

		tc.errCheck(t, e)
	}
}

func TestParseEndpoint(t *testing.T) {
	testCases := []struct {
		endpoint string
		protocol string
		addr     string
		errCheck func(t *testing.T, err error)
	}{
		{
			endpoint: fmt.Sprintf("unix://%s", testUnixPath),
			protocol: "unix",
			addr:     testUnixPath,
			errCheck: func(t *testing.T, err error) {
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			endpoint: "npipe://./pipe/containerd-containerd",
			protocol: "npipe",
			addr:     `\\.\pipe\containerd-containerd`,
			errCheck: func(t *testing.T, err error) {
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			endpoint: "192.168.123.132",
			protocol: "",
			addr:     "",
			errCheck: func(t *testing.T, err error) {
				if !errors.Is(err, ErrEndpointDeprecated) {
					t.Error(err)
				}
			},
		},
		{
			endpoint: "https://myaccount.blob.core.windows.net/mycontainer/myblob",
			protocol: "https",
			addr:     "",
			errCheck: func(t *testing.T, err error) {
				if !errors.Is(err, ErrProtocolNotSupported) {
					t.Error(err)
				}
			},
		},
		{
			endpoint: "unix://  ",
			protocol: "",
			addr:     "",
			errCheck: func(t *testing.T, err error) {
				as := &url.Error{}
				if !errors.As(err, &as) {
					t.Error(err)
				}
			},
		},
	}
	for _, tc := range testCases {
		p, a, e := ParseEndpoint(tc.endpoint)

		if p != tc.protocol || a != tc.addr {
			t.Errorf("Test fails")
		}

		tc.errCheck(t, e)
	}
}
