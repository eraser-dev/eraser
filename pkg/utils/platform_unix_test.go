//go:build !windows

package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/eraser-dev/eraser/api/unversioned"
)

// The rendezvous is not the only place a write can block: once the 64 KiB pipe
// buffer fills, a reader that has opened the FIFO and then stopped draining
// holds the writer in Write, which no open deadline covers.
//
// This is Unix-only on purpose. The same scenario is not reachable through the
// socket implementation: a single Write of 64 MiB to a stalled peer was measured
// completing in 11ms on Windows, because the OS accepts the whole overlapped
// send regardless of size.
func TestWriteImagesPipeHonoursCancellationWhileBlockedOnAStalledReader(t *testing.T) {
	// Closing a FIFO does not wake an already-blocked write outside Linux, so
	// elsewhere this would sit here until the deadline rather than fail fast.
	if runtime.GOOS != "linux" {
		t.Skipf("closing a blocked FIFO write is only guaranteed to unblock it on linux, not %s", runtime.GOOS)
	}

	path := filepath.Join(shortTempDir(t), "stalled")

	// well past the 64 KiB pipe buffer, so the write cannot simply complete
	images := make([]unversioned.Image, 120000)
	for i := range images {
		images[i] = unversioned.Image{ImageID: fmt.Sprintf("sha256:%060d", i)}
	}

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() { errCh <- WriteImagesPipe(ctx, path, images) }()

	stallReader(t, path)

	// The reader is attached, so the open has returned and the writer has moved
	// on to filling the buffer. There is no way to observe "blocked in Write"
	// directly, so give it a moment to get there -- otherwise this degrades into
	// the rendezvous case already covered in handoff_test.go.
	time.Sleep(500 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("WriteImagesPipe = %v, want context.Canceled", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("WriteImagesPipe ignored cancellation while blocked on a stalled reader")
	}
}

// stallReader opens the FIFO for reading and never reads from it. Opening is
// what releases the writer's blocked open, so the writer proceeds into Write and
// stops once the pipe buffer is full.
func stallReader(t *testing.T, path string) {
	t.Helper()

	// the writer creates the FIFO, so it may not exist yet
	deadline := time.Now().Add(10 * time.Second)
	for {
		//nolint:gosec // G304: opening the test's own pipe is the point
		f, err := os.OpenFile(path, os.O_RDONLY, 0)
		if err == nil {
			t.Cleanup(func() { _ = f.Close() })
			return
		}
		if !os.IsNotExist(err) {
			t.Fatalf("open fifo for reading: %v", err)
		}
		if time.Now().After(deadline) {
			t.Fatal("the writer never created the fifo")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

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

	conn, err := dialer(testContext(t), addr)
	if err != nil {
		t.Fatalf("dial %q: %v", addr, err)
	}
	if err := conn.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
}

func TestGetConnConnectsToALiveSocket(t *testing.T) {
	sock := filepath.Join(shortTempDir(t), "s")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := grpc.NewServer()
	go func() { _ = server.Serve(l) }()
	defer server.Stop()

	conn, err := GetConn(testContext(t), "unix://"+sock)
	if err != nil {
		t.Fatalf("GetConn: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
}

// A blocking dial used to retry every error until its context ended, and the
// workers dial with a context that never does, so these would hang rather than
// fail.
func TestGetConnFailsFastWhenTheSocketIsMissing(t *testing.T) {
	sock := filepath.Join(shortTempDir(t), "s")

	conn, err := GetConn(testContext(t), "unix://"+sock)
	if err == nil {
		_ = conn.Close()
		t.Fatal("GetConn succeeded against a socket that does not exist")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("GetConn = %v, want an error satisfying os.ErrNotExist", err)
	}
}

// A socket left behind by a runtime that has stopped refuses connections.
func TestGetConnFailsFastWhenNothingIsListening(t *testing.T) {
	sock := filepath.Join(shortTempDir(t), "s")
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: sock, Net: "unix"})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	l.SetUnlinkOnClose(false)
	if err := l.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	conn, err := GetConn(testContext(t), "unix://"+sock)
	if err == nil {
		_ = conn.Close()
		t.Fatal("GetConn succeeded against a socket nothing is listening on")
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		t.Errorf("GetConn = %v, want an error satisfying syscall.ECONNREFUSED", err)
	}
}

func TestGetConnFailsFastWhenTheSocketIsNotPermitted(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root is not subject to the socket's permissions")
	}

	sock := filepath.Join(shortTempDir(t), "s")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = l.Close() }()

	if err := os.Chmod(sock, 0); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	conn, err := GetConn(testContext(t), "unix://"+sock)
	if err == nil {
		_ = conn.Close()
		t.Fatal("GetConn succeeded against a socket it may not open")
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Errorf("GetConn = %v, want an error satisfying os.ErrPermission", err)
	}
}

// openStalledPeer completes the rendezvous and then says nothing, which is what
// leaves the reader blocked in the payload read rather than waiting to start.
func openStalledPeer(t *testing.T, path string) io.Closer {
	t.Helper()

	//nolint:gosec // G304: opening the pipe under test is the point
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("opening the endpoint as a peer: %v", err)
	}

	return f
}

func TestAwaitReportsAWriterThatSaysNothing(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "complete")
	ctx := testContext(t)

	pipe, err := CreateCompletionPipe(path)
	if err != nil {
		t.Fatalf("CreateCompletionPipe: %v", err)
	}
	defer func() { _ = pipe.Close() }()

	// opening for write completes the rendezvous, so closing without writing is
	// what a peer that died before sending looks like
	go func() {
		//nolint:gosec // G304: opening the pipe under test is the point
		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return
		}
		_ = f.Close()
	}()

	data, err := pipe.Await(ctx)
	if !errors.Is(err, ErrEmptyHandoff) {
		t.Errorf("Await = (%q, %v), want ErrEmptyHandoff", string(data), err)
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
