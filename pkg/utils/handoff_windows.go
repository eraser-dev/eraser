//go:build windows

package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"time"

	"github.com/eraser-dev/eraser/api/unversioned"
)

// Windows has no filesystem FIFO, so the handoff runs over Unix domain sockets,
// which Windows has supported since Server 2019. A socket preserves the three
// properties the FIFO implementation relies on: Accept blocks until the peer
// dials, the reader sees EOF when the peer closes, and the endpoint is a file,
// so its absence still tells the remover that no scanner is present.
//
// Responsibility is inverted relative to Unix. With a FIFO the writer creates
// the endpoint and the reader polls for it; with a socket the reader listens and
// the writer dials with retry.

// maxSocketPath is the longest usable pathname. The sun_path field is 108
// bytes and has to hold a terminating NUL, so 107 is the real ceiling.
// Exceeding it fails deep inside the syscall with an opaque error, so it is
// checked up front.
const maxSocketPath = 107

// CompletionPipe is an endpoint a peer can observe before anything is read from
// it. The scanner creates one early precisely so the remover can tell a scanner
// is present, which means the listener has to outlive its creation.
type CompletionPipe struct {
	path string
	l    net.Listener
}

// CreateCompletionPipe publishes the endpoint without waiting for a peer.
func CreateCompletionPipe(path string) (*CompletionPipe, error) {
	l, err := listen(path)
	if err != nil {
		return nil, err
	}
	return &CompletionPipe{path: path, l: l}, nil
}

// Await blocks until a peer signals completion. The payload is returned
// unvalidated so callers keep their existing handling of unexpected content.
func (p *CompletionPipe) Await() ([]byte, error) {
	conn, err := p.l.Accept()
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	return io.ReadAll(conn)
}

// Close releases the endpoint, which also unpublishes it. Callers both defer
// this and close explicitly, so repeat calls report success rather than
// net.ErrClosed.
func (p *CompletionPipe) Close() error {
	if p.l == nil {
		return nil
	}
	l := p.l
	p.l = nil
	return l.Close()
}

// WriteImagesPipe blocks until the reader is listening, then sends the list, or
// returns ctx.Err() if the context is done first.
func WriteImagesPipe(ctx context.Context, path string, images []unversioned.Image) error {
	data, err := json.Marshal(images)
	if err != nil {
		return err
	}

	conn, err := dial(ctx, path)
	if err != nil {
		return err
	}

	if _, err := conn.Write(data); err != nil {
		_ = conn.Close()
		return err
	}

	// closing is what signals end-of-message to the reader
	return conn.Close()
}

// ReadImagesPipe publishes the endpoint and waits for the writer to connect and
// finish. It returns ctx.Err() if the context is canceled while waiting.
func ReadImagesPipe(ctx context.Context, path string) ([]unversioned.Image, error) {
	l, err := listen(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = l.Close() }()

	// Accept has no context form; closing the listener is what unblocks it
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = l.Close()
		case <-done:
		}
	}()

	conn, err := l.Accept()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	data, err := io.ReadAll(conn)
	if err != nil {
		return nil, err
	}

	images := []unversioned.Image{}
	if err := json.Unmarshal(data, &images); err != nil {
		return nil, err
	}

	return images, nil
}

// WriteCompletionPipe signals a peer that this stage is done. The returned error
// satisfies os.IsNotExist when the peer never published the endpoint, which is
// how an absent scanner is detected.
func WriteCompletionPipe(ctx context.Context, path string) error {
	// Dialing a socket that is not there reports connection-refused on Windows
	// rather than ENOENT, so the filesystem is the only reliable way to tell
	// "never published" from "published but gone".
	if _, err := os.Stat(path); err != nil {
		return err
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", path)
	if err != nil {
		return err
	}

	if _, err := conn.Write([]byte(EraseCompleteMessage)); err != nil {
		_ = conn.Close()
		return err
	}

	return conn.Close()
}

func listen(path string) (net.Listener, error) {
	if len(path) > maxSocketPath {
		return nil, fmt.Errorf("socket path %q is %d bytes, over the %d byte limit", path, len(path), maxSocketPath)
	}

	// A socket left behind by an unclean exit would fail the bind, so it has to
	// go. Anything else at this path is not ours to delete: the worker runs as
	// SYSTEM and shares the volume with a scanner image we do not control.
	switch fi, err := os.Lstat(path); {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, err
	case fi.Mode()&os.ModeSocket == 0:
		return nil, fmt.Errorf("refusing to replace %q: it exists and is not a socket", path)
	default:
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	return net.Listen("unix", path)
}

// dial waits for the reader to start listening. Errors are not classified:
// Windows reports a missing socket as connection-refused, so there is no
// reliable "not yet" error to match on. Retrying on a tick mirrors the Unix
// implementation, where opening a FIFO for writing blocks until a reader
// arrives.
func dial(ctx context.Context, path string) (net.Conn, error) {
	if len(path) > maxSocketPath {
		return nil, fmt.Errorf("socket path %q is %d bytes, over the %d byte limit", path, len(path), maxSocketPath)
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var d net.Dialer
	for {
		conn, err := d.DialContext(ctx, "unix", path)
		if err == nil {
			return conn, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
