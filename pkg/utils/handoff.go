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
// endpoints in a shared volume. Each operation below was previously written out
// inline in the three worker binaries; they are gathered here so the transport
// can be swapped per platform without touching the callers.
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
	if err != nil {
		return nil, err
	}

	if err := file.Close(); err != nil {
		return nil, err
	}

	return data, nil
}

// Close releases the endpoint. It is a no-op where the endpoint is a plain
// filesystem object.
func (p *CompletionPipe) Close() error {
	return nil
}

// WriteImagesPipe publishes the endpoint and blocks until the reader connects.
func WriteImagesPipe(path string, images []unversioned.Image) error {
	data, err := json.Marshal(images)
	if err != nil {
		return err
	}

	if err := mkfifo(path, PipeMode); err != nil {
		return err
	}

	//nolint:gosec // G304: Opening pipe file is intended functionality
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}

	if _, err := file.Write(data); err != nil {
		return err
	}

	return file.Close()
}

// ReadImagesPipe waits for the endpoint to appear, then reads until the writer
// finishes. It returns ctx.Err() if the context is cancelled while waiting.
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
func WriteCompletionPipe(path string) error {
	//nolint:gosec // G304: Opening pipe file is intended functionality
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}

	if _, err := file.WriteString(EraseCompleteMessage); err != nil {
		return err
	}

	return file.Close()
}