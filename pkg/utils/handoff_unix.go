//go:build !windows

package utils

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/eraser-dev/eraser/api/unversioned"
)

// The collector, scanner and remover hand images off to each other through
// endpoints in a shared volume. On Unix those endpoints are FIFOs; see
// handoff_windows.go for the socket-based equivalent.
//
// Three properties are load-bearing and must survive any reimplementation.
// Connecting blocks until the peer is on the other end, which is what makes
// arbitrary container start order safe. The reader sees EOF when the writer
// finishes, which is what frames a message. And the absence of an endpoint is
// itself meaningful: the remover infers that the scanner is disabled from
// ENOENT on the scanner's completion endpoint.

// CompletionPipe is an endpoint a peer can observe before anything is read from
// it. The scanner creates one early precisely so the remover can tell a scanner
// is present.
type CompletionPipe struct {
	path string
}

// CreateCompletionPipe publishes the endpoint without waiting for a peer.
func CreateCompletionPipe(path string) (*CompletionPipe, error) {
	if err := mkfifo(path, PipeMode); err != nil {
		return nil, err
	}
	return &CompletionPipe{path: path}, nil
}

// Await blocks until a peer signals completion. The payload is returned
// unvalidated so callers keep their existing handling of unexpected content.
// Await blocks until a peer signals completion, or ctx is done. The payload is
// returned unvalidated so callers keep their existing handling of unexpected
// content.
func (p *CompletionPipe) Await(ctx context.Context) ([]byte, error) {
	file, err := openFifo(ctx, p.path, os.O_RDONLY)
	if err != nil {
		return nil, err
	}

	return readAndClose(ctx, file)
}

// Close releases this process's hold on the endpoint. The FIFO itself is left
// in place: the remover decides whether a scanner exists by whether the
// scanner's endpoint is on disk, so unlinking here would make a live scanner
// look absent. The socket implementation cannot keep the endpoint after Close
// because the listener owns it, which is the one lifecycle difference between
// the two.
func (p *CompletionPipe) Close() error {
	return nil
}

// WriteImagesPipe publishes the endpoint and blocks until the reader connects,
// or until ctx is done.
func WriteImagesPipe(ctx context.Context, path string, images []unversioned.Image) error {
	data, err := json.Marshal(images)
	if err != nil {
		return err
	}

	if err := mkfifo(path, PipeMode); err != nil {
		return err
	}

	file, err := openFifo(ctx, path, os.O_WRONLY)
	if err != nil {
		return err
	}

	return writeAndClose(ctx, file, data)
}

// writeAndClose writes the payload and closes, which is what frames the message
// for the reader. The open is not the only place this can block: once the pipe
// buffer fills, a reader that stops draining blocks the write too, so the
// watcher closes the file to unblock it.
//
// That last part is Linux behavior. Closing a FIFO does not wake a write that
// is already blocked on Darwin, so there cancellation is only observed once the
// reader drains enough for the write to finish. Eraser's workers run on Linux
// and Windows nodes, so this file is built for every non-Windows target but
// only guaranteed on Linux.
func writeAndClose(ctx context.Context, file *os.File, payload []byte) error {
	done := make(chan struct{})
	closedByWatcher := make(chan bool, 1)

	go func() {
		select {
		case <-ctx.Done():
			_ = file.Close()
			closedByWatcher <- true
		case <-done:
			closedByWatcher <- false
		}
	}()

	_, err := file.Write(payload)

	// Joining the watcher before touching the file again is what makes the rest
	// unambiguous: once it has reported, no cancellation close can still land,
	// and whoever closed the file is known rather than guessed from the error.
	close(done)
	if <-closedByWatcher {
		return ctx.Err()
	}

	if closeErr := file.Close(); closeErr != nil && err == nil {
		err = closeErr
	}

	return err
}

// openFifo opens a FIFO, which blocks in the kernel until the other end
// arrives. The open itself is left untouched -- the rendezvous, and the
// behavior every existing deployment depends on, is exactly as before. Only
// the waiting is made interruptible, by doing it on a goroutine that hands the
// file over if anyone is still listening and closes it if not.
func openFifo(ctx context.Context, path string, flag int) (*os.File, error) {
	type opened struct {
		file *os.File
		err  error
	}

	// Unbuffered, and paired with abandoned rather than a default case. A
	// buffered channel would accept the file after the caller had already
	// returned, orphaning the descriptor; a default case would close a file the
	// caller was about to ask for, if the open won the race to this select.
	ch := make(chan opened)
	abandoned := make(chan struct{})

	go func() {
		//nolint:gosec // G304: Opening pipe file is intended functionality
		file, err := os.OpenFile(path, flag, 0)
		select {
		case ch <- opened{file: file, err: err}:
		case <-abandoned:
			if file != nil {
				_ = file.Close()
			}
		}
	}()

	select {
	case <-ctx.Done():
		close(abandoned)
		return nil, ctx.Err()
	case o := <-ch:
		return o.file, o.err
	}
}

// ReadImagesPipe waits for the endpoint to appear, then reads until the writer
// finishes. It returns ctx.Err() if the context is canceled while waiting.
func ReadImagesPipe(ctx context.Context, path string) ([]unversioned.Image, error) {
	timer := time.NewTimer(time.Second)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()

	var f *os.File
	for {
		var err error

		f, err = openFifo(ctx, path, os.O_RDONLY)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return nil, err
		}

		timer.Reset(time.Second)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			continue
		}
	}

	data, err := readAndClose(ctx, f)
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
	// Checked before the open so that an absent peer is reported as such even
	// when ctx is already done; otherwise a terminating remover could mistake a
	// disabled scanner for a cancellation, and vice versa.
	if _, err := os.Stat(path); err != nil {
		return err
	}

	file, err := openFifo(ctx, path, os.O_WRONLY)
	if err != nil {
		return err
	}

	return writeAndClose(ctx, file, []byte(EraseCompleteMessage))
}
