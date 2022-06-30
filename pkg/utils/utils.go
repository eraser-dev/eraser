package utils

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	// unixProtocol is the network protocol of unix socket.
	unixProtocol = "unix"
)

type ExclusionList struct {
	Excluded []string `json:"excluded"`
}

var (
	ErrProtocolNotSupported  = errors.New("protocol not supported")
	ErrEndpointDeprecated    = errors.New("endpoint is deprecated, please consider using full url format")
	ErrOnlySupportUnixSocket = errors.New("only support unix socket endpoint")
)

func GetAddressAndDialer(endpoint string) (string, func(ctx context.Context, addr string) (net.Conn, error), error) {
	protocol, addr, err := ParseEndpointWithFallbackProtocol(endpoint, unixProtocol)
	if err != nil {
		return "", nil, err
	}
	if protocol != unixProtocol {
		return "", nil, ErrOnlySupportUnixSocket
	}

	return addr, dial, nil
}

func dial(ctx context.Context, addr string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, unixProtocol, addr)
}

func ParseEndpointWithFallbackProtocol(endpoint string, fallbackProtocol string) (protocol string, addr string, err error) {
	if protocol, addr, err = ParseEndpoint(endpoint); err != nil && protocol == "" {
		fallbackEndpoint := fallbackProtocol + "://" + endpoint
		protocol, addr, err = ParseEndpoint(fallbackEndpoint)
		if err != nil {
			return "", "", err
		}
	}
	return protocol, addr, err
}

func ParseEndpoint(endpoint string) (string, string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", "", fmt.Errorf("error while parsing: %w", err)
	}

	switch u.Scheme {
	case "tcp":
		return "tcp", u.Host, nil
	case "unix":
		return "unix", u.Path, nil
	case "":
		return "", "", fmt.Errorf("using %q as %w", endpoint, ErrEndpointDeprecated)
	default:
		return u.Scheme, "", fmt.Errorf("%q: %w", u.Scheme, ErrProtocolNotSupported)
	}
}

func GetImageClient(ctx context.Context, socketPath string) (pb.ImageServiceClient, *grpc.ClientConn, error) {
	addr, dialer, err := GetAddressAndDialer(socketPath)
	if err != nil {
		return nil, nil, err
	}

	conn, err := grpc.DialContext(ctx, addr, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(dialer))
	if err != nil {
		return nil, nil, err
	}

	imageClient := pb.NewImageServiceClient(conn)

	return imageClient, conn, nil
}

func ListImages(ctx context.Context, images pb.ImageServiceClient) (list []*pb.Image, err error) {
	request := &pb.ListImagesRequest{Filter: nil}

	resp, err := images.ListImages(ctx, request)
	if err != nil {
		return nil, err
	}

	return resp.Images, nil
}

func ListContainers(ctx context.Context, runtime pb.RuntimeServiceClient) (list []*pb.Container, err error) {
	resp, err := runtime.ListContainers(context.Background(), new(pb.ListContainersRequest))
	if err != nil {
		return nil, err
	}
	return resp.Containers, nil
}

func GetRunningImages(containers []*pb.Container, idToTagListMap map[string][]string) map[string]string {
	// Images that are running
	// map of (digest | tag) -> digest
	runningImages := make(map[string]string)
	for _, container := range containers {
		curr := container.Image
		digest := curr.GetImage()
		runningImages[digest] = digest

		for _, tag := range idToTagListMap[digest] {
			runningImages[tag] = digest
		}
	}
	return runningImages
}

func GetNonRunningImages(runningImages map[string]string, allImages []string, idToTagListMap map[string][]string) map[string]string {
	// Images that aren't running
	// map of (digest | tag) -> digest
	nonRunningImages := make(map[string]string)

	for _, digest := range allImages {
		if _, isRunning := runningImages[digest]; !isRunning {
			nonRunningImages[digest] = digest

			for _, tag := range idToTagListMap[digest] {
				nonRunningImages[tag] = digest
			}
		}
	}

	return nonRunningImages
}
