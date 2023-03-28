package cri

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	v1 "k8s.io/cri-api/pkg/apis/runtime/v1"
	v1alpha2 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type (
	v1alpha2Client struct {
		images  v1alpha2.ImageServiceClient
		runtime v1alpha2.RuntimeServiceClient
	}
)

func (c *v1alpha2Client) ListContainers(ctx context.Context) (list []*v1.Container, err error) {
	resp, err := c.runtime.ListContainers(ctx, new(v1alpha2.ListContainersRequest))
	if err != nil {
		return nil, err
	}
	return convertContainers(resp.Containers), nil
}

func (c *v1alpha2Client) ListImages(ctx context.Context) (list []*v1.Image, err error) {
	request := &v1alpha2.ListImagesRequest{Filter: nil}

	resp, err := c.images.ListImages(ctx, request)
	if err != nil {
		return nil, err
	}

	return convertImages(resp.Images), nil
}

func (c *v1alpha2Client) DeleteImage(ctx context.Context, image string) (err error) {
	if image == "" {
		return err
	}

	request := &v1alpha2.RemoveImageRequest{Image: &v1alpha2.ImageSpec{Image: image}}

	_, err = c.images.RemoveImage(ctx, request)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return err
	}

	return nil
}

func convertContainers(list []*v1alpha2.Container) []*v1.Container {
	v1s := []*v1.Container{}

	for _, c := range list {
		v1s = append(v1s, convertContainer(c))
	}

	return v1s
}

func convertImages(list []*v1alpha2.Image) []*v1.Image {
	v1s := []*v1.Image{}

	for _, c := range list {
		v1s = append(v1s, convertImage(c))
	}

	return v1s
}

func convertContainer(c *v1alpha2.Container) *v1.Container {
	if c == nil {
		return nil
	}

	cont := &v1.Container{
		Id:           c.Id,
		PodSandboxId: c.PodSandboxId,
		ImageRef:     c.ImageRef,
		State:        v1.ContainerState(c.State),
		CreatedAt:    c.CreatedAt,
		Labels:       c.Labels,
		Annotations:  c.Annotations,
	}

	if c.Image != nil {
		cont.Image = &v1.ImageSpec{
			Image:       c.Image.Image,
			Annotations: c.Image.Annotations,
		}
	}

	if c.Metadata != nil {
		cont.Metadata = &v1.ContainerMetadata{
			Name:    c.Metadata.Name,
			Attempt: c.Metadata.Attempt,
		}
	}

	return cont
}

func convertImage(i *v1alpha2.Image) *v1.Image {
	if i == nil {
		return nil
	}

	img := &v1.Image{
		Id:          i.Id,
		RepoTags:    i.RepoTags,
		RepoDigests: i.RepoDigests,
		Size_:       i.Size_,
		Username:    i.Username,
		Pinned:      i.Pinned,
	}

	if i.Spec != nil {
		img.Spec = &v1.ImageSpec{
			Image:       i.Spec.Image,
			Annotations: i.Spec.Annotations,
		}
	}

	if i.Uid != nil {
		img.Uid = &v1.Int64Value{
			Value: i.Uid.Value,
		}
	}

	return img
}
