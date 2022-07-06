package main

import (
	"context"

	util "github.com/Azure/eraser/pkg/utils"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type (
	client struct {
		images  pb.ImageServiceClient
		runtime pb.RuntimeServiceClient
	}

	Client interface {
		listImages(context.Context) ([]*pb.Image, error)
		listContainers(context.Context) ([]*pb.Container, error)
	}
)

func (c *client) listContainers(ctx context.Context) (list []*pb.Container, err error) {
	return util.ListContainers(ctx, c.runtime)
}

func (c *client) listImages(ctx context.Context) (list []*pb.Image, err error) {
	return util.ListImages(ctx, c.images)
}
