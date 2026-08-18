package utils

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
	go func() { errCh <- WriteImagesPipe(path, want) }()

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
	go func() { errCh <- WriteCompletionPipe(path) }()

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

	err := WriteCompletionPipe(path)
	if err == nil {
		t.Fatal("expected an error writing to an endpoint nobody published")
	}
	if !os.IsNotExist(err) {
		t.Errorf("os.IsNotExist(%v) = false, want true", err)
	}
}

func TestCompletionPipeCloseIsIdempotentlySafe(t *testing.T) {
	path := filepath.Join(shortTempDir(t), "closed")

	pipe, err := CreateCompletionPipe(path)
	if err != nil {
		t.Fatalf("CreateCompletionPipe: %v", err)
	}
	if err := pipe.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
