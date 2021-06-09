package main

import (
	"context"
	"fmt"

	"github.com/containerd/containerd"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func getContainerdImages() error {

	// namespace k8s.io in aks, default otherwise
	client, err := containerd.New("/run/containerd/containerd.sock", containerd.WithDefaultNamespace("k8s.io"))
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()

	list, err := client.ListImages(ctx)

	for _, elm := range list {
		fmt.Println(elm.Name())
	}

	return nil
}

func getDockerImages() error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	images, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return err
	}

	for _, image := range images {
		fmt.Println(image.ID)
	}

	return nil

}

func main() {

	getDockerImages()
	getContainerdImages()

}
