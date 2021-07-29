package main

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

func TestParseEndpointWithFallBackProtocol(t *testing.T) {
	var testCases = []struct {
		endpoint         string
		fallbackProtocol string
		protocol         string
		addr             string
		err              error
	}{
		{
			endpoint:         "unix:///run/containerd/containerd.sock",
			fallbackProtocol: "unix",
			protocol:         "unix",
			addr:             "/run/containerd/containerd.sock",
			err:              nil,
		},
		{
			endpoint:         "192.168.123.132",
			fallbackProtocol: "unix",
			protocol:         "unix",
			addr:             "",
			err:              nil,
		},
		{
			endpoint:         "tcp://localhost:8080",
			fallbackProtocol: "unix",
			protocol:         "tcp",
			addr:             "localhost:8080",
			err:              nil,
		},
	}

	for _, tc := range testCases {
		p, a, e := parseEndpointWithFallbackProtocol(tc.endpoint, tc.fallbackProtocol)
		if p != tc.protocol || a != tc.addr || !errors.Is(e, tc.err) {
			t.Errorf("Test fails")
		}
	}

}

func TestParseEndpoint(t *testing.T) {
	var testCases = []struct {
		endpoint string
		protocol string
		addr     string
		err      error
	}{
		{
			endpoint: "unix:///run/containerd/containerd.sock",
			protocol: "unix",
			addr:     "/run/containerd/containerd.sock",
			err:      nil,
		},
		{
			endpoint: "192.168.123.132",
			protocol: "",
			addr:     "",
			err:      ErrEndpointDeprecated,
		},
		{
			endpoint: "https://myaccount.blob.core.windows.net/mycontainer/myblob",
			protocol: "https",
			addr:     "",
			err:      ErrProtocolNotSupported,
		},
	}
	for _, tc := range testCases {
		p, a, e := parseEndpoint(tc.endpoint)
		if p != tc.protocol || a != tc.addr || !errors.Is(e, tc.err) {
			t.Errorf("Test fails")
		}
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
			err:      ErrProtocolNotSupported,
		},
		{
			endpoint: "tcp://localhost:8080",
			addr:     "",
			err:      ErrOnlySupportUnixSocket,
		},
	}

	for _, tc := range testCases {
		a, _, e := GetAddressAndDialer(tc.endpoint)
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
		RepoDigests: []string{"docker.io/ashnam/remove_images@sha256:d93d3d3073797258ef06c39e2dce9782c5c8a2315359337448e140c14423928e"},
	}

	// containers
	container1 = pb.Container{
		Id:           "7eb07fbb43e86a6114fb3b382339176117bc377cff89d5466210cbf2b101d4cb",
		PodSandboxId: "07adb7d1a87e7f79490f29846658f90456709c2b754dcffacad843d87033b2ef",
		Metadata:     &pb.ContainerMetadata{Name: "azure-ip-masq-agent", Attempt: 0},
		Image:        &pb.ImageSpec{Image: "sha256:8adbfa37c6320849612a5ade36bbb94ff03229a0587f026dd1e0561f196824ce", Annotations: map[string]string{}},
		ImageRef:     "sha256:8adbfa37c6320849612a5ade36bbb94ff03229a0587f026dd1e0561f196824ce",
		State:        pb.ContainerState_CONTAINER_RUNNING,
		CreatedAt:    1626446035006572211,
		Labels:       map[string]string{"io.kubernetes.container.name": "azure-ip-masq-agent", "io.kubernetes.pod.name": "azure-ip-masq-agent-wt85g", "io.kubernetes.pod.namespace": "kube-system", "io.kubernetes.pod.uid": "d668fc57-3c20-4680-93cd-af4af112b98c"},
		Annotations:  map[string]string{"io.kubernetes.container.hash": "a77771", "io.kubernetes.container.restartCount": "0", "io.kubernetes.container.terminationMessagePath": "/dev/termination-log", "io.kubernetes.container.terminationMessagePolicy": "File", "io.kubernetes.pod.terminationGracePeriod": "30"},
	}

	container2 = pb.Container{
		Id:           "36080589120ee72504484c0f407568c49531021c751bc55b3ccd5af03b8af2cb",
		PodSandboxId: "1ee4a30267a9fadd11aeb11377efc500a831fe4082e372e3191d6d31cf5e3ea1",
		Metadata:     &pb.ContainerMetadata{Name: "kube-proxy", Attempt: 0},
		Image:        &pb.ImageSpec{Image: "sha256:b4034db328056e7f4c27ab76a5b9811b0f5eaa99565194cf7c6446781e772043", Annotations: map[string]string{}},
		ImageRef:     "sha256:b4034db328056e7f4c27ab76a5b9811b0f5eaa99565194cf7c6446781e772043",
		State:        pb.ContainerState_CONTAINER_RUNNING,
		CreatedAt:    1626446037402720790,
		Labels:       map[string]string{"io.kubernetes.container.name": "kube-proxy", "io.kubernetes.pod.name": "kube-proxy-s5c67", "io.kubernetes.pod.namespace": "kube-system", "io.kubernetes.pod.uid": "f6dddad9-b772-4735-a8df-db910fdca461"},
		Annotations:  map[string]string{"io.kubernetes.container.hash": "a11f949b", "io.kubernetes.container.restartCount": "0", "io.kubernetes.container.terminationMessagePath": "/dev/termination-log", "io.kubernetes.container.terminationMessagePolicy": "File", "io.kubernetes.pod.terminationGracePeriod": "302"},
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

	for index, value := range c.images {
		for _, repotag := range value.RepoTags {
			if value.Id == image || repotag == image {
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
	return ErrImageNotRemoved
}

func testEqImages(a, b []*pb.Image) bool {
	if len(a) != len(b) {
		return false
	}
	m1 := make(map[string]bool, len(a))
	for _, i := range a {
		m1[i.Id] = true
	}
	for _, j := range b {
		if m1[j.Id] == false {
			return false
		}
	}
	return true
}

func testEqContainers(a, b []*pb.Container) bool {
	if len(a) != len(b) {
		return false
	}
	m1 := make(map[string]bool, len(a))
	for _, i := range a {
		m1[i.Id] = true
	}
	for _, j := range b {
		if m1[j.Id] == false {
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
	}

	backgroundContext, _ := context.WithTimeout(context.Background(), timeoutTest)

	for _, tc := range testCases {
		l, e := tc.imagesInput.listImages(backgroundContext)
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
	}

	backgroundContext, _ := context.WithTimeout(context.Background(), timeoutTest)

	for _, tc := range testCases {
		l, e := tc.containersInput.listContainers(backgroundContext)
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
				containers: []*pb.Container{},
				images:     []*pb.Image{&image1, &image2, &image3, &image4, &image5},
			},
			imageToDelete: "sha256:d153e49438bdcf34564a4e6b4f186658ca1168043be299106f8d6048e8617574",
			imagesOutput:  []*pb.Image{&image1, &image3, &image4, &image5},
			err:           nil,
		},
		{
			imagesInput: testClient{
				containers: []*pb.Container{},
				images:     []*pb.Image{&image1, &image2, &image3, &image4, &image5},
			},
			imageToDelete: "",
			imagesOutput:  []*pb.Image{&image1, &image2, &image3, &image4, &image5},
			err:           ErrImageEmpty,
		},
	}

	backgroundContext, _ := context.WithTimeout(context.Background(), timeoutTest)

	for _, tc := range testCases {
		e := tc.imagesInput.removeImage(backgroundContext, tc.imageToDelete)
		if testEqImages(tc.imagesInput.images, tc.imagesOutput) == false || !errors.Is(e, tc.err) {
			t.Errorf("Test fails")
		}
	}

}
