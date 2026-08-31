package utils

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eraser-dev/eraser/api/unversioned"
)

// These run against whichever implementation the platform selects: FIFOs on
// Unix, Unix domain sockets on Windows. Keeping them build-tag free is the point
// -- the two implementations have to stay behaviorally identical.

// shortTempDir keeps paths well inside the sun_path limit that applies to the
// socket implementation.
func shortTempDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "h")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	return dir
}

// testContext bounds the round trips. Both halves of a handoff block until the
// peer shows up, so on context.Background a rendezvous that never completes
// hangs until the package-wide test timeout -- ten minutes of nothing, with no
// indication of which test is stuck. A deadline turns that into a failure in
// the test that caused it.
func testContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	return ctx
}

func TestImagesHandoffRoundTrip(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "images")
	ctx := testContext(t)

	want := []unversioned.Image{
		{ImageID: "sha256:aaaa", Names: []string{"repo/one:v1"}},
		{ImageID: "sha256:bbbb", Names: []string{"repo/two:v2"}},
	}

	errCh := make(chan error, 1)
	go func() { errCh <- WriteImagesPipe(ctx, path, want) }()

	// The read has to be off the test goroutine for the deadline to mean
	// anything: once the FIFO opens or the socket accepts, the payload read
	// stops observing ctx, so a stalled peer would hang here rather than fail.
	type readResult struct {
		images []unversioned.Image
		err    error
	}
	readCh := make(chan readResult, 1)
	go func() {
		images, err := ReadImagesPipe(ctx, path)
		readCh <- readResult{images: images, err: err}
	}()

	var got readResult
	select {
	case got = <-readCh:
	case <-ctx.Done():
		t.Fatalf("ReadImagesPipe did not return within the deadline: %v", ctx.Err())
	}
	if got.err != nil {
		t.Fatalf("ReadImagesPipe: %v", got.err)
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("WriteImagesPipe: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("WriteImagesPipe did not return within the deadline: %v", ctx.Err())
	}

	if len(got.images) != len(want) {
		t.Fatalf("got %d images, want %d", len(got.images), len(want))
	}
	for i := range want {
		if got.images[i].ImageID != want[i].ImageID {
			t.Errorf("image %d = %q, want %q", i, got.images[i].ImageID, want[i].ImageID)
		}
	}
}

func TestCompletionHandoffRoundTrip(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "complete")
	ctx := testContext(t)

	pipe, err := CreateCompletionPipe(path)
	if err != nil {
		t.Fatalf("CreateCompletionPipe: %v", err)
	}
	defer func() { _ = pipe.Close() }()

	errCh := make(chan error, 1)
	go func() { errCh <- WriteCompletionPipe(ctx, path) }()

	type awaited struct {
		data []byte
		err  error
	}

	// Await takes no context, so it cannot be given the deadline directly; the
	// select is what enforces it.
	awaitCh := make(chan awaited, 1)
	go func() {
		data, err := pipe.Await()
		awaitCh <- awaited{data: data, err: err}
	}()

	var got awaited
	select {
	case got = <-awaitCh:
	case <-ctx.Done():
		t.Fatalf("Await did not return within the deadline: %v", ctx.Err())
	}
	if got.err != nil {
		t.Fatalf("Await: %v", got.err)
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("WriteCompletionPipe: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("WriteCompletionPipe did not return within the deadline: %v", ctx.Err())
	}

	if string(got.data) != EraseCompleteMessage {
		t.Errorf("payload = %q, want %q", string(got.data), EraseCompleteMessage)
	}
}

// The remover infers "the scanner is disabled" from this error, so it has to
// keep satisfying os.IsNotExist on both platforms.
func TestWriteCompletionPipeAbsentPeerIsNotExist(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "no-such-peer")

	err := WriteCompletionPipe(context.Background(), path)
	if err == nil {
		t.Fatal("expected an error writing to an endpoint nobody published")
	}
	if !os.IsNotExist(err) {
		t.Errorf("os.IsNotExist(%v) = false, want true", err)
	}
}

// The pre-open Stat exists so this precedence holds: an endpoint nobody
// published reports IsNotExist even when the caller is already shutting down.
// Left to a select, the two would race and "the scanner is disabled" would
// become indistinguishable from "we are terminating".
func TestWriteCompletionPipeAbsentPeerBeatsACanceledContext(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "no-such-peer")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := WriteCompletionPipe(ctx, path)
	if err == nil {
		t.Fatal("expected an error writing to an endpoint nobody published")
	}
	if !os.IsNotExist(err) {
		t.Errorf("os.IsNotExist(%v) = false, want true", err)
	}
}

// The whole point of taking a context: a worker whose peer never arrives has to
// be able to give up, on either platform.
func TestWriteImagesPipeHonoursACanceledContext(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "never-read")

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() { errCh <- WriteImagesPipe(ctx, path, []unversioned.Image{{ImageID: "sha256:aaaa"}}) }()

	// nothing ever reads the endpoint, so the write is still waiting
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("WriteImagesPipe = %v, want context.Canceled", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("WriteImagesPipe ignored the canceled context")
	}
}

func TestCompletionPipeCloseIsIdempotentlySafe(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "closed")

	pipe, err := CreateCompletionPipe(path)
	if err != nil {
		t.Fatalf("CreateCompletionPipe: %v", err)
	}

	// the scanner defers Close and the collector also closes explicitly
	if err := pipe.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := pipe.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}
