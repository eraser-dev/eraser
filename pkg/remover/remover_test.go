package main

import (
	"testing"

	v1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

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
		tc := tc
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

			_, err := removeImages(client, tc.remove)
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
