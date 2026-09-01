//go:build !windows

package utils

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
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

func TestUnixDialerConnects(t *testing.T) {
	// short base dir: sun_path is limited to ~108 bytes
	dir, err := os.MkdirTemp("", "d")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	sock := filepath.Join(dir, "s")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = l.Close() }()
	go func() {
		if c, err := l.Accept(); err == nil {
			_ = c.Close()
		}
	}()

	addr, dialer, err := getAddressAndDialer("unix://" + sock)
	if err != nil {
		t.Fatalf("getAddressAndDialer: %v", err)
	}

	conn, err := dialer(context.Background(), addr)
	if err != nil {
		t.Fatalf("dial %q: %v", addr, err)
	}
	if err := conn.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
}

func TestMkfifoCreatesNamedPipe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fifo")
	if err := mkfifo(path, PipeMode); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		t.Errorf("mode = %v, want a named pipe", fi.Mode())
	}
}
