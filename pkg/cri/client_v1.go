package cri

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	v1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type (
	v1Client struct {
		images  v1.ImageServiceClient
		runtime v1.RuntimeServiceClient
	}
)

func (c *v1Client) ListContainers(ctx context.Context) (list []*v1.Container, err error) {
	resp, err := c.runtime.ListContainers(ctx, new(v1.ListContainersRequest))
	if err != nil {
		return nil, err
	}
	return resp.Containers, nil
}

func (c *v1Client) ListImages(ctx context.Context) (list []*v1.Image, err error) {
	request := &v1.ListImagesRequest{Filter: nil}

	resp, err := c.images.ListImages(ctx, request)
	if err != nil {
		return nil, err
	}

	return resp.Images, nil
}

func (c *v1Client) DeleteImage(ctx context.Context, image string) (err error) {
	if image == "" {
		return err
	}

	request := &v1.RemoveImageRequest{Image: &v1.ImageSpec{Image: image}}

	_, err = c.images.RemoveImage(ctx, request)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return err
	}

	return nil
}
