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

func TestImagesHandoffRoundTrip(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "images")

	want := []unversioned.Image{
		{ImageID: "sha256:aaaa", Names: []string{"repo/one:v1"}},
		{ImageID: "sha256:bbbb", Names: []string{"repo/two:v2"}},
	}

	errCh := make(chan error, 1)
	go func() { errCh <- WriteImagesPipe(context.Background(), path, want) }()

	got, err := ReadImagesPipe(context.Background(), path)
	if err != nil {
		t.Fatalf("ReadImagesPipe: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("WriteImagesPipe: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("got %d images, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ImageID != want[i].ImageID {
			t.Errorf("image %d = %q, want %q", i, got[i].ImageID, want[i].ImageID)
		}
	}
}

func TestCompletionHandoffRoundTrip(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "complete")

	pipe, err := CreateCompletionPipe(path)
	if err != nil {
		t.Fatalf("CreateCompletionPipe: %v", err)
	}
	defer func() { _ = pipe.Close() }()

	errCh := make(chan error, 1)
	go func() { errCh <- WriteCompletionPipe(context.Background(), path) }()

	data, err := pipe.Await()
	if err != nil {
		t.Fatalf("Await: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("WriteCompletionPipe: %v", err)
	}

	if string(data) != EraseCompleteMessage {
		t.Errorf("payload = %q, want %q", string(data), EraseCompleteMessage)
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
