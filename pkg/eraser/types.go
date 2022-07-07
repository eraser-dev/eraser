package main

import (
	"context"

	util "github.com/Azure/eraser/pkg/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		deleteImage(context.Context, string) error
	}
)

func (c *client) listContainers(ctx context.Context) (list []*pb.Container, err error) {
	return util.ListContainers(ctx, c.runtime)
}

func (c *client) listImages(ctx context.Context) (list []*pb.Image, err error) {
	return util.ListImages(ctx, c.images)
}

func (c *client) deleteImage(ctx context.Context, image string) (err error) {
	if image == "" {
		return err
	}

	request := &pb.RemoveImageRequest{Image: &pb.ImageSpec{Image: image}}

	_, err = c.images.RemoveImage(ctx, request)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return err
	}

	return nil
}
