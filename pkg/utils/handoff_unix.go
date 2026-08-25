//go:build !windows

package utils

import (
	"context"
	"encoding/json"
	"io"
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
func (p *CompletionPipe) Await() ([]byte, error) {
	//nolint:gosec // G304: Opening pipe file is intended functionality
	file, err := os.OpenFile(p.path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(file)
	if closeErr := file.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}

	return data, nil
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

	file, err := openForWrite(ctx, path)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	if closeErr := file.Close(); closeErr != nil && err == nil {
		err = closeErr
	}

	return err
}

// openForWrite opens a FIFO for writing, which blocks in the kernel until a
// reader arrives. The open itself is left untouched -- the rendezvous, and the
// behavior every existing deployment depends on, is exactly as before. Only
// the waiting is made interruptible, by doing it on a goroutine that hands the
// file over if anyone is still listening and closes it if not.
func openForWrite(ctx context.Context, path string) (*os.File, error) {
	type opened struct {
		file *os.File
		err  error
	}

	ch := make(chan opened, 1)
	go func() {
		//nolint:gosec // G304: Opening pipe file is intended functionality
		file, err := os.OpenFile(path, os.O_WRONLY, 0)
		select {
		case ch <- opened{file: file, err: err}:
		default:
			if file != nil {
				_ = file.Close()
			}
		}
	}()

	select {
	case <-ctx.Done():
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

		//nolint:gosec // G304: Opening pipe file is intended functionality
		f, err = os.OpenFile(path, os.O_RDONLY, 0)
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

	data, err := io.ReadAll(f)
	if closeErr := f.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
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

	file, err := openForWrite(ctx, path)
	if err != nil {
		return err
	}

	_, err = file.WriteString(EraseCompleteMessage)
	if closeErr := file.Close(); closeErr != nil && err == nil {
		err = closeErr
	}

	return err
}
