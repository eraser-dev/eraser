package main

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/Azure/eraser/pkg/util"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

func TestParseEndpointWithFallBackProtocol(t *testing.T) {
	var testCases = []struct {
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
	var testCases = []struct {
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
				if !errors.Is(err, util.ErrEndpointDeprecated) {
					t.Error(err)
				}
			},
		},
		{
			endpoint: "https://myaccount.blob.core.windows.net/mycontainer/myblob",
			protocol: "https",
			addr:     "",
			errCheck: func(t *testing.T, err error) {
				if !errors.Is(err, util.ErrProtocolNotSupported) {
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
	var testCases = []struct {
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
			err:      util.ErrProtocolNotSupported,
		},
		{
			endpoint: "tcp://localhost:8080",
			addr:     "",
			err:      util.ErrOnlySupportUnixSocket,
		},
	}

	for _, tc := range testCases {
		a, _, e := util.GetAddressAndDialer(tc.endpoint)
		if a != tc.addr || !errors.Is(e, tc.err) {
			t.Errorf("Test fails")
		}
	}
}

type testClient struct {
	containers []*pb.Container
	images     []*pb.Image
}

var (
	ErrImageNotRemoved = errors.New("image not removed")
	ErrImageEmpty      = errors.New("unable to remove empty image")
	timeoutTest        = 10 * time.Second

	// images
	image1 = pb.Image{
		Id:       "sha256:ccd78eb0f420877b5513f61bf470dd379d8e8672671115d65c6f69d1c4261f87",
		RepoTags: []string{"mcr.microsoft.com/aks/acc/sgx-webhook:0.6"},
	}
	image2 = pb.Image{
		Id:       "sha256:d153e49438bdcf34564a4e6b4f186658ca1168043be299106f8d6048e8617574",
		RepoTags: []string{"mcr.microsoft.com/containernetworking/azure-npm:v1.2.1"},
	}
	image3 = pb.Image{
		Id:       "sha256:8adbfa37c6320849612a5ade36bbb94ff03229a0587f026dd1e0561f196824ce",
		RepoTags: []string{"mcr.microsoft.com/oss/kubernetes/ip-masq-agent:v2.5.0.4"},
	}
	image4 = pb.Image{
		Id:          "sha256:b4034db328056e7f4c27ab76a5b9811b0f5eaa99565194cf7c6446781e772043",
		RepoTags:    []string{"mcr.microsoft.com/oss/kubernetes/kube-proxy:v1.19.11-hotfix.20210526"},
		RepoDigests: []string{"mcr.microsoft.com/oss/kubernetes/kube-proxy@sha256:a64d3538b72905b07356881314755b02db3675ff47ee2bcc49dd7be856e285d5"},
	}
	image5 = pb.Image{
		Id:          "sha256:fd46ec1af6de89db1714a243efa1e35c4408f5a5b9df9c653dd70db1ee95522b",
		RepoTags:    []string{},
		RepoDigests: []string{"docker.io/ghcr.io/azure/remove_images@sha256:d93d3d3073797258ef06c39e2dce9782c5c8a2315359337448e140c14423928e"},
	}

	// containers
	container1 = pb.Container{
		Id:       "7eb07fbb43e86a6114fb3b382339176117bc377cff89d5466210cbf2b101d4cb",
		Image:    &pb.ImageSpec{Image: "sha256:8adbfa37c6320849612a5ade36bbb94ff03229a0587f026dd1e0561f196824ce", Annotations: map[string]string{}},
		ImageRef: "sha256:8adbfa37c6320849612a5ade36bbb94ff03229a0587f026dd1e0561f196824ce",
	}

	container2 = pb.Container{
		Id:       "36080589120ee72504484c0f407568c49531021c751bc55b3ccd5af03b8af2cb",
		Image:    &pb.ImageSpec{Image: "sha256:b4034db328056e7f4c27ab76a5b9811b0f5eaa99565194cf7c6446781e772043", Annotations: map[string]string{}},
		ImageRef: "sha256:b4034db328056e7f4c27ab76a5b9811b0f5eaa99565194cf7c6446781e772043",
	}
)

func (c *testClient) listImages(ctx context.Context) (list []*pb.Image, err error) {
	images := make([]*pb.Image, len(c.images))
	copy(images, c.images)
	return images, nil
}

func (c *testClient) listContainers(ctx context.Context) (list []*pb.Container, err error) {
	containers := make([]*pb.Container, len(c.containers))
	copy(containers, c.containers)
	return containers, nil
}

func (c *testClient) removeImageFromSlice(index int) {
	s := c.images
	s = append(s[:index], s[index+1:]...)
	c.images = s
}

func (c *testClient) removeImage(ctx context.Context, image string) (err error) {
	if image == "" {
		return ErrImageEmpty
	}
	containersImageNames := make(map[string]bool, len(c.containers))
	for _, container := range c.containers {
		containersImageNames[container.ImageRef] = true
	}
	for index, value := range c.images {
		for _, repotag := range value.RepoTags {
			if (value.Id == image || repotag == image) && containersImageNames[value.Id] == false {
				c.removeImageFromSlice(index)
				return nil
			}
		}

		for _, repodigest := range value.RepoDigests {
			if repodigest == image && containersImageNames[value.Id] == false {
				c.removeImageFromSlice(index)
				return nil
			}
		}
	}
	return ErrImageNotRemoved
}

func testEqImages(a, b []*pb.Image) bool {
	if len(a) != len(b) {
		return false
	}
	auxMap := make(map[string]bool, len(a))
	for _, i := range a {
		auxMap[i.Id] = true
	}
	for _, j := range b {
		if auxMap[j.Id] == false {
			return false
		}
	}
	return true
}

func testEqContainers(a, b []*pb.Container) bool {
	if len(a) != len(b) {
		return false
	}
	auxMap := make(map[string]bool, len(a))
	for _, i := range a {
		auxMap[i.Id] = true
	}
	for _, j := range b {
		if auxMap[j.Id] == false {
			return false
		}
	}
	return true
}

func TestListImages(t *testing.T) {
	var testCases = []struct {
		imagesInput  testClient
		imagesOutput []*pb.Image
		err          error
	}{
		{
			imagesInput: testClient{
				containers: []*pb.Container{},
				images:     []*pb.Image{&image1, &image2, &image3, &image4, &image5},
			},
			imagesOutput: []*pb.Image{&image1, &image2, &image3, &image4, &image5},
			err:          nil,
		},
		{
			imagesInput: testClient{
				containers: []*pb.Container{},
				images:     []*pb.Image{},
			},
			imagesOutput: []*pb.Image{},
			err:          nil,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutTest)
	defer cancel()

	for _, tc := range testCases {
		l, e := tc.imagesInput.listImages(ctx)
		if testEqImages(l, tc.imagesOutput) == false || !errors.Is(e, tc.err) {
			t.Errorf("Test fails")
		}
	}

}

func TestListContainers(t *testing.T) {
	var testCases = []struct {
		containersInput testClient
		containerOutput []*pb.Container
		err             error
	}{
		{
			containersInput: testClient{
				containers: []*pb.Container{&container1, &container2},
				images:     []*pb.Image{},
			},
			containerOutput: []*pb.Container{&container1, &container2},
			err:             nil,
		},
		{
			containersInput: testClient{
				containers: []*pb.Container{},
				images:     []*pb.Image{},
			},
			containerOutput: []*pb.Container{},
			err:             nil,
		},
		{
			containersInput: testClient{
				containers: []*pb.Container{},
				images:     []*pb.Image{&image1},
			},
			containerOutput: []*pb.Container{},
			err:             nil,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutTest)
	defer cancel()

	for _, tc := range testCases {
		l, e := tc.containersInput.listContainers(ctx)
		if testEqContainers(l, tc.containerOutput) == false || !errors.Is(e, tc.err) {
			t.Errorf("Test fails")
		}
	}

}

func TestRemoveImage(t *testing.T) {
	var testCases = []struct {
		imagesInput   testClient
		imageToDelete string
		imagesOutput  []*pb.Image
		err           error
	}{
		{
			imagesInput: testClient{
				containers: []*pb.Container{&container1, &container2},
				images:     []*pb.Image{&image1, &image2, &image3, &image4, &image5},
			},
			imageToDelete: "sha256:ccd78eb0f420877b5513f61bf470dd379d8e8672671115d65c6f69d1c4261f87",
			imagesOutput:  []*pb.Image{&image2, &image3, &image4, &image5},
			err:           nil,
		},
		{
			imagesInput: testClient{
				containers: []*pb.Container{&container1, &container2},
				images:     []*pb.Image{&image1, &image2, &image3, &image4, &image5},
			},
			imageToDelete: "mcr.microsoft.com/containernetworking/azure-npm:v1.2.1",
			imagesOutput:  []*pb.Image{&image1, &image3, &image4, &image5},
			err:           nil,
		},
		{
			imagesInput: testClient{
				containers: []*pb.Container{&container1, &container2},
				images:     []*pb.Image{&image1, &image2, &image3, &image4, &image5},
			},
			imageToDelete: "docker.io/ghcr.io/azure/remove_images@sha256:d93d3d3073797258ef06c39e2dce9782c5c8a2315359337448e140c14423928e",
			imagesOutput:  []*pb.Image{&image1, &image2, &image3, &image4},
			err:           nil,
		},
		{
			imagesInput: testClient{
				containers: []*pb.Container{&container1, &container2},
				images:     []*pb.Image{&image1, &image2, &image3, &image4, &image5},
			},
			imageToDelete: "",
			imagesOutput:  []*pb.Image{&image1, &image2, &image3, &image4, &image5},
			err:           ErrImageEmpty,
		},
		{
			imagesInput: testClient{
				containers: []*pb.Container{&container1, &container2},
				images:     []*pb.Image{},
			},
			imageToDelete: "",
			imagesOutput:  []*pb.Image{},
			err:           ErrImageEmpty,
		},
		{
			imagesInput: testClient{
				containers: []*pb.Container{&container1, &container2},
				images:     []*pb.Image{&image2},
			},
			imageToDelete: "sha256:d153e49438bdcf34564a4e6b4f186658ca1168043be299106f8d6048e8617574",
			imagesOutput:  []*pb.Image{},
			err:           nil,
		},
		{
			imagesInput: testClient{
				containers: []*pb.Container{&container1, &container2},
				images:     []*pb.Image{&image1, &image2, &image3, &image4, &image5},
			},
			imageToDelete: "hellothere",
			imagesOutput:  []*pb.Image{&image1, &image2, &image3, &image4, &image5},
			err:           ErrImageNotRemoved,
		},
		{
			imagesInput: testClient{
				containers: []*pb.Container{&container1, &container2},
				images:     []*pb.Image{&image1, &image2, &image3, &image4, &image5},
			},
			imageToDelete: "sha256:8adbfa37c6320849612a5ade36bbb94ff03229a0587f026dd1e0561f196824ce",
			imagesOutput:  []*pb.Image{&image1, &image2, &image3, &image4, &image5},
			err:           ErrImageNotRemoved,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutTest)
	cancel()

	for _, tc := range testCases {
		e := tc.imagesInput.removeImage(ctx, tc.imageToDelete)
		if testEqImages(tc.imagesInput.images, tc.imagesOutput) == false || !errors.Is(e, tc.err) {
			t.Errorf("Test fails")
		}
	}

}
