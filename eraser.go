package main

import (
	"context"
	"fmt"

	"github.com/containerd/containerd"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func getContainerdImages() error {

	client, err := containerd.New("/run/containerd/containerd.sock", containerd.WithDefaultNamespace("default"))
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer client.Close()

	ctx := context.Background()

	list, err := client.ListImages(ctx)

	for _, elm := range list {
		fmt.Println(elm.Target())
	}

	return nil
}

func getDockerImages() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Println(err)
	}

	images, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		fmt.Println(err)
	}

	for _, image := range images {
		fmt.Println(image.ID)
	}
}

func main() {
	fmt.Println("Docker Images:")
	getDockerImages()

	fmt.Println("Containerd Images:")
	getContainerdImages()

}
