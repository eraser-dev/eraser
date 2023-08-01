package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eraser-dev/eraser/pkg/cri"
	v1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type testLogger interface {
	Logf(format string, args ...interface{})
}

type testClient struct {
	containers []*v1.Container
	images     []*v1.Image
	t          testLogger
}

var (
	_ cri.Remover = &testClient{}

	errImageNotRemoved = errors.New("image not removed")
	errImageEmpty      = errors.New("unable to remove empty image")
	timeoutTest        = 10 * time.Second

	image1 = v1.Image{
		Id:       "sha256:ccd78eb0f420877b5513f61bf470dd379d8e8672671115d65c6f69d1c4261f87",
		RepoTags: []string{"mcr.microsoft.com/aks/acc/sgx-webhook:0.6"},
	}
	image2 = v1.Image{
		Id:       "sha256:d153e49438bdcf34564a4e6b4f186658ca1168043be299106f8d6048e8617574",
		RepoTags: []string{"mcr.microsoft.com/containernetworking/azure-npm:v1.2.1"},
	}
	image3 = v1.Image{
		Id:       "sha256:8adbfa37c6320849612a5ade36bbb94ff03229a0587f026dd1e0561f196824ce",
		RepoTags: []string{"mcr.microsoft.com/oss/kubernetes/ip-masq-agent:v2.5.0.4"},
	}
	image4 = v1.Image{
		Id:          "sha256:b4034db328056e7f4c27ab76a5b9811b0f5eaa99565194cf7c6446781e772043",
		RepoTags:    []string{"mcr.microsoft.com/oss/kubernetes/kube-proxy:v1.19.11-hotfix.20210526"},
		RepoDigests: []string{"mcr.microsoft.com/oss/kubernetes/kube-proxy@sha256:a64d3538b72905b07356881314755b02db3675ff47ee2bcc49dd7be856e285d5"},
	}
	image5 = v1.Image{
		Id:          "sha256:fd46ec1af6de89db1714a243efa1e35c4408f5a5b9df9c653dd70db1ee95522b",
		RepoTags:    []string{},
		RepoDigests: []string{"docker.io/aldaircoronel/remove_images@sha256:d93d3d3073797258ef06c39e2dce9782c5c8a2315359337448e140c14423928e"},
	}

	container1 = v1.Container{
		Id:       "7eb07fbb43e86a6114fb3b382339176117bc377cff89d5466210cbf2b101d4cb",
		Image:    &v1.ImageSpec{Image: "sha256:8adbfa37c6320849612a5ade36bbb94ff03229a0587f026dd1e0561f196824ce", Annotations: map[string]string{}},
		ImageRef: "sha256:8adbfa37c6320849612a5ade36bbb94ff03229a0587f026dd1e0561f196824ce",
	}
	container2 = v1.Container{
		Id:       "36080589120ee72504484c0f407568c49531021c751bc55b3ccd5af03b8af2cb",
		Image:    &v1.ImageSpec{Image: "sha256:b4034db328056e7f4c27ab76a5b9811b0f5eaa99565194cf7c6446781e772043", Annotations: map[string]string{}},
		ImageRef: "sha256:b4034db328056e7f4c27ab76a5b9811b0f5eaa99565194cf7c6446781e772043",
	}
)

func (c *testClient) logf(format string, args ...interface{}) {
	if c.t == nil {
		return
	}
	c.t.Logf(format, args...)
}

func (c *testClient) ListImages(_ context.Context) (list []*v1.Image, err error) {
	images := make([]*v1.Image, len(c.images))
	copy(images, c.images)
	return images, nil
}

func (c *testClient) ListContainers(_ context.Context) (list []*v1.Container, err error) {
	containers := make([]*v1.Container, len(c.containers))
	copy(containers, c.containers)
	return containers, nil
}

func (c *testClient) removeImageFromSlice(index int) {
	s := c.images
	s = append(s[:index], s[index+1:]...)
	c.images = s
}

func (c *testClient) DeleteImage(_ context.Context, image string) (err error) {
	c.logf("DeleteImage: %s", image)
	if image == "" {
		return errImageEmpty
	}
	for index, value := range c.images {
		if value.Id == image {
			c.removeImageFromSlice(index)
			return nil
		}
		for _, repotag := range value.RepoTags {
			if repotag == image {
				c.removeImageFromSlice(index)
				return nil
			}
		}

		for _, repodigest := range value.RepoDigests {
			if repodigest == image {
				c.removeImageFromSlice(index)
				return nil
			}
		}
	}
	return errImageNotRemoved
}

func testEqImages(a, b []*v1.Image) bool {
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

func testEqContainers(a, b []*v1.Container) bool {
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

func TestTestClient(t *testing.T) {
	t.Run("ListImages", testListImages)
	t.Run("ListContainers", testListContainers)
	t.Run("DeleteImage", testDeleteImage)
}

func testListImages(t *testing.T) {
	testCases := []struct {
		imagesInput  testClient
		imagesOutput []*v1.Image
		err          error
	}{
		{
			imagesInput: testClient{
				containers: []*v1.Container{},
				images:     []*v1.Image{&image1, &image2, &image3, &image4, &image5},
			},
			imagesOutput: []*v1.Image{&image1, &image2, &image3, &image4, &image5},
			err:          nil,
		},
		{
			imagesInput: testClient{
				containers: []*v1.Container{},
				images:     []*v1.Image{},
			},
			imagesOutput: []*v1.Image{},
			err:          nil,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutTest)
	defer cancel()

	for _, tc := range testCases {
		l, e := tc.imagesInput.ListImages(ctx)
		if testEqImages(l, tc.imagesOutput) == false || !errors.Is(e, tc.err) {
			t.Errorf("Test fails")
		}
	}
}

func testListContainers(t *testing.T) {
	testCases := []struct {
		containersInput testClient
		containerOutput []*v1.Container
		err             error
	}{
		{
			containersInput: testClient{
				containers: []*v1.Container{&container1, &container2},
				images:     []*v1.Image{},
			},
			containerOutput: []*v1.Container{&container1, &container2},
			err:             nil,
		},
		{
			containersInput: testClient{
				containers: []*v1.Container{},
				images:     []*v1.Image{},
			},
			containerOutput: []*v1.Container{},
			err:             nil,
		},
		{
			containersInput: testClient{
				containers: []*v1.Container{},
				images:     []*v1.Image{&image1},
			},
			containerOutput: []*v1.Container{},
			err:             nil,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutTest)
	defer cancel()

	for _, tc := range testCases {
		l, e := tc.containersInput.ListContainers(ctx)
		if testEqContainers(l, tc.containerOutput) == false || !errors.Is(e, tc.err) {
			t.Errorf("Test fails")
		}
	}
}

func testDeleteImage(t *testing.T) {
	testCases := []struct {
		imagesInput   testClient
		imageToDelete string
		imagesOutput  []*v1.Image
		err           error
	}{
		{
			imagesInput: testClient{
				containers: []*v1.Container{&container1, &container2},
				images:     []*v1.Image{&image1, &image2, &image3, &image4, &image5},
			},
			imageToDelete: "sha256:ccd78eb0f420877b5513f61bf470dd379d8e8672671115d65c6f69d1c4261f87",
			imagesOutput:  []*v1.Image{&image2, &image3, &image4, &image5},
			err:           nil,
		},
		{
			imagesInput: testClient{
				containers: []*v1.Container{&container1, &container2},
				images:     []*v1.Image{&image1, &image2, &image3, &image4, &image5},
			},
			imageToDelete: "mcr.microsoft.com/containernetworking/azure-npm:v1.2.1",
			imagesOutput:  []*v1.Image{&image1, &image3, &image4, &image5},
			err:           nil,
		},
		{
			imagesInput: testClient{
				containers: []*v1.Container{&container1, &container2},
				images:     []*v1.Image{&image1, &image2, &image3, &image4, &image5},
			},
			imageToDelete: "docker.io/aldaircoronel/remove_images@sha256:d93d3d3073797258ef06c39e2dce9782c5c8a2315359337448e140c14423928e",
			imagesOutput:  []*v1.Image{&image1, &image2, &image3, &image4},
			err:           nil,
		},
		{
			imagesInput: testClient{
				containers: []*v1.Container{&container1, &container2},
				images:     []*v1.Image{&image1, &image2, &image3, &image4, &image5},
			},
			imageToDelete: "",
			imagesOutput:  []*v1.Image{&image1, &image2, &image3, &image4, &image5},
			err:           errImageEmpty,
		},
		{
			imagesInput: testClient{
				containers: []*v1.Container{&container1, &container2},
				images:     []*v1.Image{},
			},
			imageToDelete: "",
			imagesOutput:  []*v1.Image{},
			err:           errImageEmpty,
		},
		{
			imagesInput: testClient{
				containers: []*v1.Container{&container1, &container2},
				images:     []*v1.Image{&image2},
			},
			imageToDelete: "sha256:d153e49438bdcf34564a4e6b4f186658ca1168043be299106f8d6048e8617574",
			imagesOutput:  []*v1.Image{},
			err:           nil,
		},
		{
			imagesInput: testClient{
				containers: []*v1.Container{&container1, &container2},
				images:     []*v1.Image{&image1, &image2, &image3, &image4, &image5},
			},
			imageToDelete: "hellothere",
			imagesOutput:  []*v1.Image{&image1, &image2, &image3, &image4, &image5},
			err:           errImageNotRemoved,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutTest)
	defer cancel()

	for i, tc := range testCases {
		e := tc.imagesInput.DeleteImage(ctx, tc.imageToDelete)
		if testEqImages(tc.imagesInput.images, tc.imagesOutput) == false || !errors.Is(e, tc.err) {
			t.Errorf("Test fails: %d", i)
		}
	}
}
