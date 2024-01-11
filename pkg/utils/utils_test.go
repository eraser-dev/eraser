package utils

import (
	"errors"
	"fmt"
	"net/url"
	"testing"
)

func TestParseEndpointWithFallBackProtocol(t *testing.T) {
	testCases := []struct {
		endpoint         string
		fallbackProtocol string
		protocol         string
		addr             string
		errCheck         func(t *testing.T, err error)
	}{
		{
			endpoint:         fmt.Sprintf("unix://%s", CRIPath),
			fallbackProtocol: "unix",
			protocol:         "unix",
			addr:             CRIPath,
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
			endpoint: fmt.Sprintf("unix://%s", CRIPath),
			protocol: "unix",
			addr:     CRIPath,
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

func TestGetAddressAndDialer(t *testing.T) {
	testCases := []struct {
		endpoint string
		addr     string
		err      error
	}{
		{
			endpoint: fmt.Sprintf("unix://%s", CRIPath),
			addr:     CRIPath,
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
			t.Errorf("Test fails")
		}
	}
}
