package main

import (
	"context"
	"errors"
	"testing"

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

var ErrImageNotRemoved = errors.New("image not removed")
var ErrImageEmpty = errors.New("unable to remove empty image")

type testClient struct {
	containers []*pb.Container
	images     []*pb.Image
}

func (client *testClient) listImages(ctx context.Context) (list []*pb.Image, err error) {
	images := make([]*pb.Image, len(client.images))
	copy(images, client.images)
	return images, nil
}

func (client *testClient) listContainers(ctx context.Context) (list []*pb.Container, err error) {
	containers := make([]*pb.Container, len(client.containers))
	copy(containers, client.containers)
	return containers, nil
}

func (client *testClient) removeImgFromSlice(index int) {
	s := client.images
	s = append(s[:index], s[index+1:]...)
	client.images = s
}

func (client *testClient) removeImage(ctx context.Context, image string) (err error) {
	if image == "" {
		return ErrImageEmpty
	}

	for index, value := range client.images {
		for _, repotag := range value.RepoTags {
			if value.Id == image || repotag == image {
				client.removeImgFromSlice(index)
				return nil
			}
		}

		for _, repodigest := range value.RepoDigests {
			if repodigest == image {
				client.removeImgFromSlice(index)
				return nil
			}
		}
	}
	return ErrImageNotRemoved
}

func TestRemoveVulnerableImages(t *testing.T) {
	image1 := pb.Image{
		Id:       "sha256:ccd78eb0f420877b5513f61bf470dd379d8e8672671115d65c6f69d1c4261f87",
		RepoTags: []string{"mcr.microsoft.com/aks/acc/sgx-webhook:0.6"},
	}
	image2 := pb.Image{
		Id:       "sha256:d153e49438bdcf34564a4e6b4f186658ca1168043be299106f8d6048e8617574",
		RepoTags: []string{"mcr.microsoft.com/containernetworking/azure-npm:v1.2.1"},
	}
	image3 := pb.Image{
		Id:       "sha256:8adbfa37c6320849612a5ade36bbb94ff03229a0587f026dd1e0561f196824ce",
		RepoTags: []string{"mcr.microsoft.com/oss/kubernetes/ip-masq-agent:v2.5.0.4"},
	}
	image4 := pb.Image{
		Id:          "sha256:b4034db328056e7f4c27ab76a5b9811b0f5eaa99565194cf7c6446781e772043",
		RepoTags:    []string{"mcr.microsoft.com/oss/kubernetes/kube-proxy:v1.19.11-hotfix.20210526"},
		RepoDigests: []string{"mcr.microsoft.com/oss/kubernetes/kube-proxy@sha256:a64d3538b72905b07356881314755b02db3675ff47ee2bcc49dd7be856e285d5"},
	}
	image5 := pb.Image{
		Id:          "sha256:fd46ec1af6de89db1714a243efa1e35c4408f5a5b9df9c653dd70db1ee95522b",
		RepoDigests: []string{"docker.io/ashnam/remove_images@sha256:d93d3d3073797258ef06c39e2dce9782c5c8a2315359337448e140c14423928e"},
	}

	// containers
	container1 := pb.Container{
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

	container2 := pb.Container{
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

	client := testClient{
		containers: []*pb.Container{&container1, &container2},
		images:     []*pb.Image{&image1, &image2, &image3, &image4, &image5},
	}

	removeVulnerableImages(&client, "", "")
}
