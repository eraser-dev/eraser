//go:build windows

package utils

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
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

// The guard exists to replace an opaque syscall failure with a clear message,
// so the boundary it enforces has to be the real one.
func TestSocketPathLimitBoundary(t *testing.T) {
	dir := shortTempDir(t)

	pathOfLen := func(n int) string {
		return filepath.Join(dir, strings.Repeat("a", n-len(dir)-1))
	}

	atLimit := pathOfLen(maxSocketPath)
	l, err := listen(atLimit)
	if err != nil {
		t.Fatalf("listen(%d-byte path) = %v, want success", len(atLimit), err)
	}
	_ = l.Close()

	overLimit := pathOfLen(maxSocketPath + 1)
	if _, err := listen(overLimit); err == nil {
		t.Errorf("listen(%d-byte path) = nil, want the guard to reject it", len(overLimit))
	}
}

// The worker runs as SYSTEM and shares the volume with a scanner image we do
// not control, so an occupied endpoint path is a reason to stop rather than to
// start deleting.
func TestListenRefusesToReplaceANonSocket(t *testing.T) {
	dir := shortTempDir(t)
	path := filepath.Join(dir, "occupied")

	if err := os.WriteFile(path, []byte("not a socket"), 0o600); err != nil {
		t.Fatal(err)
	}

	if l, err := listen(path); err == nil {
		_ = l.Close()
		t.Fatal("listen replaced a regular file, want an error")
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("the file was removed anyway: %v", err)
	}
}

// A socket the previous run failed to clean up must still be reclaimable,
// otherwise a crashed worker would poison the endpoint for every retry.
func TestListenReclaimsAStaleSocket(t *testing.T) {
	dir := shortTempDir(t)
	path := filepath.Join(dir, "stale")

	// Go unlinks the socket on Close, so the only endpoint left on disk is one
	// nobody closed. SetUnlinkOnClose reproduces that without crashing a
	// process: the file stays, the listener does not.
	stale, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	stale.SetUnlinkOnClose(false)
	if err := stale.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("the endpoint should still be on disk: %v", err)
	}

	l, err := listen(path)
	if err != nil {
		t.Fatalf("listen over a stale socket: %v", err)
	}
	_ = l.Close()
}

// The case above is indistinguishable from this one by mode alone, and taking
// the path from a peer that is still listening would strand it silently.
func TestListenRefusesALiveSocket(t *testing.T) {
	dir := shortTempDir(t)
	path := filepath.Join(dir, "live")

	live, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = live.Close() }()

	if l, err := listen(path); err == nil {
		_ = l.Close()
		t.Fatal("listen replaced a live socket, want an error")
	}

	if _, err := os.Lstat(path); err != nil {
		t.Errorf("the live endpoint was removed anyway: %v", err)
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
