package cri

import (
	"context"
	"fmt"

	"github.com/eraser-dev/eraser/pkg/utils"
	"google.golang.org/grpc"
	v1 "k8s.io/cri-api/pkg/apis/runtime/v1"
	v1alpha2 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	RuntimeV1       runtimeVersion = "v1"
	RuntimeV1Alpha2 runtimeVersion = "v1alpha2"
)

type (
	Collector interface {
		ListImages(context.Context) ([]*v1.Image, error)
		ListContainers(context.Context) ([]*v1.Container, error)
	}

	Remover interface {
		Collector
		DeleteImage(context.Context, string) error
	}

	runtimeTryFunc func(context.Context, *grpc.ClientConn) (string, error)
)

func NewCollectorClient(socketPath string) (Collector, error) {
	return NewRemoverClient(socketPath)
}

func NewRemoverClient(socketPath string) (Remover, error) {
	ctx := context.Background()

	conn, err := utils.GetConn(ctx, socketPath)
	if err != nil {
		return nil, err
	}

	return newClientWithFallback(ctx, conn)
}

func newClientWithFallback(ctx context.Context, conn *grpc.ClientConn) (Remover, error) {
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
	if err != nil {
		return "", err
	}

	return respv1Alpha2.RuntimeApiVersion, err
}

func tryV1(ctx context.Context, conn *grpc.ClientConn) (string, error) {
	runtimeClient := v1.NewRuntimeServiceClient(conn)
	req := v1.VersionRequest{}

	resp, err := runtimeClient.Version(ctx, &req)
	if err != nil {
		return "", err
	}

	return resp.RuntimeApiVersion, err
}

func getClientFromRuntimeVersion(conn *grpc.ClientConn, runtimeAPIVersion string) (Remover, error) {
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
