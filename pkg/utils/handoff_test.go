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

	// Read off the test goroutine so the deadline holds even if the code under
	// test stops observing ctx: a regression there would hang the package rather
	// than fail this test.
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

	data, err := pipe.Await(ctx)
	if err != nil {
		t.Fatalf("Await: %v", err)
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("WriteCompletionPipe: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("WriteCompletionPipe did not return within the deadline: %v", ctx.Err())
	}

	if string(data) != EraseCompleteMessage {
		t.Errorf("payload = %q, want %q", string(data), EraseCompleteMessage)
	}
}

// The collector keeps signal notification registered across Await now, so the
// read has to be interruptible after the peer has arrived, not only while
// waiting for one. A peer that connects and then goes quiet is what the pod
// looks like when the manager's deployment is deleted mid-run.
func TestAwaitHonoursCancellationWhileBlockedOnAStalledPeer(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "complete")

	pipe, err := CreateCompletionPipe(path)
	if err != nil {
		t.Fatalf("CreateCompletionPipe: %v", err)
	}
	defer func() { _ = pipe.Close() }()

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		_, err := pipe.Await(ctx)
		errCh <- err
	}()

	// rendezvous, then hold the endpoint open without sending anything
	peer := openStalledPeer(t, path)
	defer func() { _ = peer.Close() }()

	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Await = %v, want context.Canceled", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Await ignored the canceled context while blocked on the payload read")
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
