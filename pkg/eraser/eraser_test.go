package main

import (
	"errors"
	"net/url"
	"testing"

	util "github.com/Azure/eraser/pkg/utils"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

var (
	errProtocolNotSupported  = errors.New("protocol not supported")
	errEndpointDeprecated    = errors.New("endpoint is deprecated, please consider using full url format")
	errOnlySupportUnixSocket = errors.New("only support unix socket endpoint")
)

func TestParseEndpointWithFallBackProtocol(t *testing.T) {
	testCases := []struct {
		endpoint         string
		fallbackProtocol string
		protocol         string
		addr             string
		errCheck         func(t *testing.T, err error)
	}{
		{
			endpoint:         "unix:///run/containerd/containerd.sock",
			fallbackProtocol: "unix",
			protocol:         "unix",
			addr:             "/run/containerd/containerd.sock",
			errCheck: func(t *testing.T, err error) {
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			endpoint:         "192.168.123.132",
			fallbackProtocol: "unix",
			protocol:         "unix",
			addr:             "",
			errCheck: func(t *testing.T, err error) {
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			endpoint:         "tcp://localhost:8080",
			fallbackProtocol: "unix",
			protocol:         "tcp",
			addr:             "localhost:8080",
			errCheck: func(t *testing.T, err error) {
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			endpoint:         "  ",
			fallbackProtocol: "unix",
			protocol:         "",
			addr:             "",
			errCheck: func(t *testing.T, err error) {
				as := &url.Error{}
				if !errors.As(err, &as) {
					t.Error(err)
				}
			},
		},
	}

	for _, tc := range testCases {
		p, a, e := util.ParseEndpointWithFallbackProtocol(tc.endpoint, tc.fallbackProtocol)

		if p != tc.protocol || a != tc.addr {
			t.Errorf("Test fails")
		}

		tc.errCheck(t, e)
	}
}

func TestParseEndpoint(t *testing.T) {
	testCases := []struct {
		endpoint string
		protocol string
		addr     string
		errCheck func(t *testing.T, err error)
	}{
		{
			endpoint: "unix:///run/containerd/containerd.sock",
			protocol: "unix",
			addr:     "/run/containerd/containerd.sock",
			errCheck: func(t *testing.T, err error) {
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			endpoint: "192.168.123.132",
			protocol: "",
			addr:     "",
			errCheck: func(t *testing.T, err error) {
				if !errors.Is(err, errEndpointDeprecated) {
					t.Error(err)
				}
			},
		},
		{
			endpoint: "https://myaccount.blob.core.windows.net/mycontainer/myblob",
			protocol: "https",
			addr:     "",
			errCheck: func(t *testing.T, err error) {
				if !errors.Is(err, errProtocolNotSupported) {
					t.Error(err)
				}
			},
		},
		{
			endpoint: "unix://  ",
			protocol: "",
			addr:     "",
			errCheck: func(t *testing.T, err error) {
				as := &url.Error{}
				if !errors.As(err, &as) {
					t.Error(err)
				}
			},
		},
	}
	for _, tc := range testCases {
		p, a, e := util.ParseEndpoint(tc.endpoint)

		if p != tc.protocol || a != tc.addr {
			t.Errorf("Test fails")
		}

		tc.errCheck(t, e)
	}
}

func TestGetAddressAndDialer(t *testing.T) {
	testCases := []struct {
		endpoint string
		addr     string
		err      error
	}{
		{
			endpoint: "unix:///var/run/dockershim.sock",
			addr:     "/var/run/dockershim.sock",
			err:      nil,
		},
		{
			endpoint: "localhost:8080",
			addr:     "",
			err:      errProtocolNotSupported,
		},
		{
			endpoint: "tcp://localhost:8080",
			addr:     "",
			err:      errOnlySupportUnixSocket,
		},
	}

	for _, tc := range testCases {
		a, _, e := util.GetAddressAndDialer(tc.endpoint)
		if a != tc.addr || !errors.Is(e, tc.err) {
			t.Errorf("Test fails")
		}
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
		tc := tc
		t.Run(k, func(t *testing.T) {
			client := &testClient{t: t}
			added := make(map[string]struct{})
			running := make(map[string]struct{})
			for j := range tc.running {
				client.containers = append(client.containers, &pb.Container{
					Image: &pb.ImageSpec{Image: tc.running[j]},
				})
				client.images = append(client.images, &pb.Image{Id: tc.running[j]})
				added[tc.running[j]] = struct{}{}
				running[tc.running[j]] = struct{}{}
			}

			for j := range tc.cached {
				if _, ok := added[tc.cached[j]]; !ok {
					client.images = append(client.images, &pb.Image{Id: tc.cached[j]})
				}
			}

			err := removeImages(client, tc.remove)
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
