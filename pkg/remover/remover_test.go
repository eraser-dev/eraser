package main

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// The loop guard runs before each deletion, so on the last one there is no
// later iteration to notice the caller has gone. Without a check on the error
// path, that deletion fails and the run still returns success.
func TestRemoveImagesSurfacesCancellationDuringTheFinalDeletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	client := &testClient{
		t: t,
		images: []*v1.Image{
			{Id: "aaa"},
			{Id: "bbb"},
		},
	}
	client.beforeDelete = func(_ context.Context, image string) error {
		if image == "bbb" {
			cancel()
		}
		return nil
	}

	removed, err := removeImages(ctx, client, []string{"aaa", "bbb"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("removeImages = %v, want context.Canceled", err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1 -- the first deletion did succeed", removed)
	}
}

// nonRunningImages holds an entry for the ID, for every tag and for every
// digest, so a prune reaches one image several times over. A deletion that
// fails is what exposes it: a successful one is already unreachable a second
// time, whereas a failure used to be retried per alias, each retry starting
// the per-image budget again.
func TestRemoveImagesPrunesAnImageOnceAcrossItsAliases(t *testing.T) {
	client := &testClient{
		t: t,
		images: []*v1.Image{
			{
				Id:          "sha256:aaaa",
				RepoTags:    []string{"repo/one:v1"},
				RepoDigests: []string{"repo/one@sha256:bbbb"},
			},
		},
	}

	attempts := 0
	client.beforeDelete = func(context.Context, string) error {
		attempts++
		return errImageNotRemoved
	}

	if _, err := removeImages(context.Background(), client, []string{"*"}); err != nil {
		t.Fatalf("removeImages: %v", err)
	}
	if attempts != 1 {
		t.Errorf("DeleteImage attempted %d times for one image, want 1", attempts)
	}
}

// The point of this PR is that the budget is per deletion rather than per run,
// and every other test here would still pass if the timeout moved back outside
// the loops or the parent context were handed straight to the runtime. This one
// looks at the deadlines themselves.
func TestRemoveImagesGivesEachDeletionItsOwnDeadline(t *testing.T) {
	client := &testClient{
		t: t,
		images: []*v1.Image{
			{Id: "aaa"},
			{Id: "bbb"},
			{Id: "ccc"},
		},
	}

	var deadlines []time.Time
	client.beforeDelete = func(ctx context.Context, image string) error {
		d, ok := ctx.Deadline()
		if !ok {
			t.Errorf("DeleteImage(%s) got a context with no deadline", image)
			return nil
		}
		deadlines = append(deadlines, d)

		// guarantees the clock moves between deletions, so a shared deadline is
		// distinguishable from a fresh one rather than merely unlikely
		time.Sleep(2 * time.Millisecond)
		return nil
	}

	// the parent has no deadline of its own, so anything seen above came from
	// the per-deletion budget
	if _, err := removeImages(context.Background(), client, []string{"aaa", "bbb", "ccc"}); err != nil {
		t.Fatalf("removeImages: %v", err)
	}

	if len(deadlines) != 3 {
		t.Fatalf("saw %d deletions, want 3", len(deadlines))
	}
	for i, d := range deadlines {
		if until := time.Until(d); until <= 0 || until > deleteTimeout {
			t.Errorf("deletion %d has %v left, want a fresh budget of at most %v", i, until, deleteTimeout)
		}
	}
	if !deadlines[2].After(deadlines[0]) {
		t.Error("every deletion shared one deadline, so the budget is per run rather than per image")
	}
}

// A caller that has gone away should stop the loop, not work through the rest
// of the list. Previously the shared context expired mid-run and every
// remaining image logged an instant failure while the run still reported
// partial success.
func TestRemoveImagesStopsWhenTheCallerIsGone(t *testing.T) {
	client := &testClient{
		t: t,
		images: []*v1.Image{
			{Id: "aaa"},
			{Id: "bbb"},
			{Id: "ccc"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	removed, err := removeImages(ctx, client, []string{"aaa", "bbb", "ccc"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("removeImages = %v, want context.Canceled", err)
	}
	if removed != 0 {
		t.Errorf("removed = %d, want 0 once the caller is gone", removed)
	}
}

func TestRemoveImages(t *testing.T) {
	type testCase struct {
		running   []string
		cached    []string
		remove    []string
		expect    []string
		shouldErr bool
	}

	// In these cases "running" are automatically populated into the list of cached images just to remove uneccessary duplication
	// "Prune" in the test case names refers to using "*" to remove all non-running images.
	cases := map[string]testCase{
		"No images at all":                       {},
		"Images to remove but no images on node": {remove: []string{"image1", "image2"}},
		"No images to remove but images on node": {cached: []string{"image1", "image2"}, expect: []string{"image1", "image2"}},
		"Remove subset of images":                {cached: []string{"image1", "image2", "image3"}, remove: []string{"image1", "image2"}, expect: []string{"image3"}},
		"Remove all images explicitly":           {cached: []string{"image1", "image2", "image3"}, remove: []string{"image1", "image2", "image3"}, expect: []string{}},
		"Remove single running image":            {running: []string{"image1"}, remove: []string{"image1"}, expect: []string{"image1"}},
		"Remove multiple running images":         {cached: []string{"image1"}, running: []string{"image2", "image3"}, remove: []string{"image2", "image3"}, expect: []string{"image1", "image2", "image3"}},
		"Remove all images by prune":             {cached: []string{"image1", "image2", "image3"}, remove: []string{"*"}, expect: []string{}},
		"Prune and explicit image running=false": {cached: []string{"image1", "image2", "image3"}, remove: []string{"*", "image2"}, expect: []string{}},
		"Prune and explicit image running=true":  {running: []string{"image1"}, cached: []string{"image2", "image3"}, remove: []string{"*", "image2"}, expect: []string{"image1"}},
	}

	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			client := &testClient{t: t}
			added := make(map[string]struct{})
			running := make(map[string]struct{})
			for j := range tc.running {
				client.containers = append(client.containers, &v1.Container{
					Image: &v1.ImageSpec{Image: tc.running[j]},
				})
				client.images = append(client.images, &v1.Image{Id: tc.running[j]})
				added[tc.running[j]] = struct{}{}
				running[tc.running[j]] = struct{}{}
			}

			for j := range tc.cached {
				if _, ok := added[tc.cached[j]]; !ok {
					client.images = append(client.images, &v1.Image{Id: tc.cached[j]})
				}
			}

			_, err := removeImages(context.Background(), client, tc.remove)
			if tc.shouldErr && err == nil {
				t.Fatal("expected error, got none")
			}
			if !tc.shouldErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			images := make(map[string]struct{})

			for k := range client.images {
				images[client.images[k].Id] = struct{}{}
			}

			if len(tc.expect) != len(images) {
				t.Fatalf("unexpected imaages remaining: expected: %v, got: %v", tc.expect, images)
			}

			for j := range tc.expect {
				if _, ok := images[tc.expect[j]]; !ok {
					t.Fatalf("expected image to still exist: %s", tc.expect[j])
				}
			}
			for j := range tc.remove {
				if _, ok := running[tc.remove[j]]; ok {
					// Skip checking if image still exists if it is running
					continue
				}
				if _, ok := images[tc.remove[j]]; ok {
					t.Fatalf("expected image to be removed: %s", tc.remove[j])
				}
			}
		})
	}
}
