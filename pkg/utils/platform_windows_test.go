//go:build windows

package utils

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	assertListenRefuses(t, path)
}

// Nothing of ours outlives the pod at these paths, so both cases below are
// anomalies rather than something to tidy up -- and they are indistinguishable
// on disk anyway, which is why the distinction is no longer attempted.
func TestListenRefusesAnEndpointThatAlreadyExists(t *testing.T) {
	t.Run("left behind by a dead listener", func(t *testing.T) {
		path := filepath.Join(shortTempDir(t), "stale")

		// Go unlinks the socket on Close, so the only endpoint left on disk is
		// one nobody closed. SetUnlinkOnClose reproduces that without having to
		// crash a process: the file stays, the listener does not.
		stale, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
		if err != nil {
			t.Fatal(err)
		}
		stale.SetUnlinkOnClose(false)
		if err := stale.Close(); err != nil {
			t.Fatal(err)
		}

		assertListenRefuses(t, path)
	})

	t.Run("still being served", func(t *testing.T) {
		path := filepath.Join(shortTempDir(t), "live")

		live, err := net.Listen("unix", path)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = live.Close() }()

		assertListenRefuses(t, path)
	})
}

func assertListenRefuses(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("the endpoint should be on disk before the call: %v", err)
	}

	if l, err := listen(path); err == nil {
		_ = l.Close()
		t.Fatal("listen bound over an existing endpoint, want an error")
	}

	if _, err := os.Lstat(path); err != nil {
		t.Errorf("the endpoint was removed anyway: %v", err)
	}
}

// These endpoints serve exactly one connection and the peer does not reconnect,
// so a connect that says nothing means the writer died or was canceled before
// sending. Reporting it beats waiting for a second connection that is never
// coming.
func TestAwaitReportsAConnectThatSaysNothing(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "complete")
	ctx := testContext(t)

	pipe, err := CreateCompletionPipe(path)
	if err != nil {
		t.Fatalf("CreateCompletionPipe: %v", err)
	}
	defer func() { _ = pipe.Close() }()

	conn, err := net.DialTimeout("unix", path, time.Second)
	if err != nil {
		t.Fatalf("connecting to the endpoint: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	type awaited struct {
		data []byte
		err  error
	}
	awaitCh := make(chan awaited, 1)
	go func() {
		data, err := pipe.Await()
		awaitCh <- awaited{data: data, err: err}
	}()

	select {
	case got := <-awaitCh:
		if !errors.Is(got.err, ErrEmptyHandoff) {
			t.Errorf("Await = (%q, %v), want ErrEmptyHandoff", string(got.data), got.err)
		}
	case <-ctx.Done():
		t.Fatalf("Await did not return within the deadline: %v", ctx.Err())
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

	conn, err := dialer(testContext(t), addr)
	if err != nil {
		t.Fatalf("dial %q: %v", addr, err)
	}
	if err := conn.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
}
