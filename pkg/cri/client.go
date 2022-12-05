package cri

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/eraser/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	v1 "k8s.io/cri-api/pkg/apis/runtime/v1"
	v1alpha2 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	RuntimeV1       runtimeVersion = "v1"
	RuntimeV1Alpha2 runtimeVersion = "v1alpha2"
)

type (
	runtimeVersion string

	errors []error

	v1alpha2Client struct {
		images  v1alpha2.ImageServiceClient
		runtime v1alpha2.RuntimeServiceClient
	}

	v1Client struct {
		images  v1.ImageServiceClient
		runtime v1.RuntimeServiceClient
	}

	Collector interface {
		ListImages(context.Context) ([]*v1.Image, error)
		ListContainers(context.Context) ([]*v1.Container, error)
	}

	Eraser interface {
		Collector
		DeleteImage(context.Context, string) error
	}

	runtimeTryFunc func(context.Context, *grpc.ClientConn) (string, error)
)

func (errs errors) Error() string {
	s := make([]string, 0, len(errs))
	for _, err := range errs {
		s = append(s, err.Error())
	}

	return strings.Join(s, "\n")
}

func (errs *errors) Append(err error) {
	if err == nil {
		return
	}
	*errs = append(*errs, err)
}

func NewCollectorClient(socketPath string) (Collector, error) {
	return NewEraserClient(socketPath)
}

func NewEraserClient(socketPath string) (Eraser, error) {
	ctx := context.Background()

	conn, err := utils.GetConn(ctx, socketPath)
	if err != nil {
		return nil, err
	}

	return newClientWithFallback(conn, ctx)
}

func newClientWithFallback(conn *grpc.ClientConn, ctx context.Context) (Eraser, error) {
	errs := new(errors)
	funcs := []runtimeTryFunc{tryV1, tryV1Alpha2}

	for _, f := range funcs {
		version, err := f(ctx, conn)
		if err != nil {
			errs.Append(err)
			continue
		}

		client, err := getClientFromRuntimeVersion(conn, version)
		if err != nil {
			errs.Append(err)
			continue
		}

		return client, nil
	}

	return nil, errs
}

func tryV1Alpha2(ctx context.Context, conn *grpc.ClientConn) (string, error) {
	runtimeClientV1Alpha2 := v1alpha2.NewRuntimeServiceClient(conn)
	req2 := v1alpha2.VersionRequest{}
	respv1Alpha2, err := runtimeClientV1Alpha2.Version(ctx, &req2)
	return respv1Alpha2.RuntimeApiVersion, err
}

func tryV1(ctx context.Context, conn *grpc.ClientConn) (string, error) {
	runtimeClient := v1.NewRuntimeServiceClient(conn)
	req := v1.VersionRequest{}
	resp, err := runtimeClient.Version(ctx, &req)
	return resp.RuntimeApiVersion, err
}

func (c *v1alpha2Client) ListContainers(ctx context.Context) (list []*v1.Container, err error) {
	resp, err := c.runtime.ListContainers(context.Background(), new(v1alpha2.ListContainersRequest))
	if err != nil {
		return nil, err
	}
	return ConvertContainers(resp.Containers), nil
}

func (c *v1alpha2Client) ListImages(ctx context.Context) (list []*v1.Image, err error) {
	request := &v1alpha2.ListImagesRequest{Filter: nil}

	resp, err := c.images.ListImages(ctx, request)
	if err != nil {
		return nil, err
	}

	return ConvertImages(resp.Images), nil
}

func (c *v1Client) ListContainers(ctx context.Context) (list []*v1.Container, err error) {
	resp, err := c.runtime.ListContainers(context.Background(), new(v1.ListContainersRequest))
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

func getClientFromRuntimeVersion(conn *grpc.ClientConn, runtimeAPIVersion string) (Eraser, error) {
	switch runtimeAPIVersion {
	case string(RuntimeV1):
		imageClient := v1.NewImageServiceClient(conn)
		runtimeClient := v1.NewRuntimeServiceClient(conn)
		return &v1Client{
			images:  imageClient,
			runtime: runtimeClient,
		}, nil
	case string(RuntimeV1Alpha2):
		runtimeClient := v1alpha2.NewRuntimeServiceClient(conn)
		imageClient := v1alpha2.NewImageServiceClient(conn)
		return &v1alpha2Client{
			images:  imageClient,
			runtime: runtimeClient,
		}, nil
	}

	return nil, fmt.Errorf("unrecognized CRI version: '%s'", runtimeAPIVersion)
}

func ConvertContainers(list []*v1alpha2.Container) []*v1.Container {
	v1s := []*v1.Container{}

	for _, c := range list {
		v1s = append(v1s, convertContainer(c))
	}

	return v1s
}

func ConvertImages(list []*v1alpha2.Image) []*v1.Image {
	v1s := []*v1.Image{}

	for _, c := range list {
		v1s = append(v1s, convertImage(c))
	}

	return v1s
}

func JoinErrors(errs []error) error {
	s := make([]string, 0, len(errs))
	for _, err := range errs {
		s = append(s, err.Error())
	}

	return fmt.Errorf("%s", strings.Join(s, "\n"))
}

func convertContainer(c *v1alpha2.Container) *v1.Container {
	return &v1.Container{
		Id:           c.Id,
		PodSandboxId: c.PodSandboxId,
		Metadata: &v1.ContainerMetadata{
			Name:    c.Metadata.Name,
			Attempt: c.Metadata.Attempt,
		},
		Image: &v1.ImageSpec{
			Image:       c.Image.Image,
			Annotations: c.Image.Annotations,
		},
		ImageRef:    c.ImageRef,
		State:       v1.ContainerState(c.State),
		CreatedAt:   c.CreatedAt,
		Labels:      c.Labels,
		Annotations: c.Annotations,
	}
}

func convertImage(i *v1alpha2.Image) *v1.Image {
	return &v1.Image{
		Id:          i.Id,
		RepoTags:    i.RepoTags,
		RepoDigests: i.RepoDigests,
		Size_:       i.Size_,
		Uid: &v1.Int64Value{
			Value: i.Uid.Value,
		},
		Username: i.Username,
		Spec: &v1.ImageSpec{
			Image:       i.Spec.Image,
			Annotations: i.Spec.Annotations,
		},
		Pinned: i.Pinned,
	}
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
