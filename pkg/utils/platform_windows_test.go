//go:build windows

package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/Microsoft/go-winio"
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
			// kubelet spells it with four slashes: empty host, already-UNC path
			endpoint: "npipe:////./pipe/containerd-containerd",
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

func TestNpipeDialerConnects(t *testing.T) {
	name := fmt.Sprintf("eraser-test-%d", os.Getpid())

	l, err := winio.ListenPipe(`\\.\pipe\`+name, nil)
	if err != nil {
		t.Fatalf("listen pipe: %v", err)
	}
	defer func() { _ = l.Close() }()
	go func() {
		if c, err := l.Accept(); err == nil {
			_ = c.Close()
		}
	}()

	addr, dialer, err := getAddressAndDialer("npipe://./pipe/" + name)
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
