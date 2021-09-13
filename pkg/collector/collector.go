package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"time"

	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	// unixProtocol is the network protocol of unix socket.
	unixProtocol = "unix"
	apiPath      = "apis/eraser.sh/v1alpha1"
)

var (
	// Timeout  of connecting to server (default: 10s)
	timeout                  = 10 * time.Second
	ErrProtocolNotSupported  = errors.New("protocol not supported")
	ErrEndpointDeprecated    = errors.New("endpoint is deprecated, please consider using full url format")
	ErrOnlySupportUnixSocket = errors.New("only support unix socket endpoint")
)

type client struct {
	images  pb.ImageServiceClient
	runtime pb.RuntimeServiceClient
}

type Client interface {
	listImages(context.Context) ([]*pb.Image, error)
}

func (c *client) listImages(ctx context.Context) (list []*pb.Image, err error) {
	request := &pb.ListImagesRequest{Filter: nil}

	resp, err := c.images.ListImages(ctx, request)
	if err != nil {
		return nil, err
	}

	return resp.Images, nil
}

func GetAddressAndDialer(endpoint string) (string, func(ctx context.Context, addr string) (net.Conn, error), error) {
	protocol, addr, err := parseEndpointWithFallbackProtocol(endpoint, unixProtocol)
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

func parseEndpointWithFallbackProtocol(endpoint string, fallbackProtocol string) (protocol string, addr string, err error) {
	if protocol, addr, err = parseEndpoint(endpoint); err != nil && protocol == "" {
		fallbackEndpoint := fallbackProtocol + "://" + endpoint
		protocol, addr, err = parseEndpoint(fallbackEndpoint)
		if err != nil {
			return "", "", err
		}
	}
	return protocol, addr, err
}

func parseEndpoint(endpoint string) (string, string, error) {
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

func getImageClient(ctx context.Context, socketPath string) (pb.ImageServiceClient, *grpc.ClientConn, error) {
	addr, dialer, err := GetAddressAndDialer(socketPath)
	if err != nil {
		return nil, nil, err
	}

	conn, err := grpc.DialContext(ctx, addr, grpc.WithBlock(), grpc.WithInsecure(), grpc.WithContextDialer(dialer))
	if err != nil {
		return nil, nil, err
	}

	imageClient := pb.NewImageServiceClient(conn)

	return imageClient, conn, nil
}

func logError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func writeListImagesToCollectorCR(clientSet *kubernetes.Clientset, c Client, socketPath string) (err error) {
	backgroundContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	images, err := c.listImages(backgroundContext)
	logError(err)

	// list of images id's
	imagesIds := make([]string, 0, len(images))

	// map of sha id and repotags
	idRepoMap := make(map[string][]string)

	for _, image := range images {
		imagesIds = append(imagesIds, image.Id)
		idRepoMap[image.Id] = image.repoTags
	}

	return nil
}

func main() {
	runtimePtr := flag.String("runtime", "docker", "container runtime")
	flag.Parse()

	var socketPath string

	switch runtime := *runtimePtr; runtime {
	case "docker":
		socketPath = "unix:///var/run/dockershim.sock"
	case "containerd":
		socketPath = "unix:///run/containerd/containerd.sock"
	case "cri-o":
		socketPath = "unix:///var/run/crio/crio.sock"
	default:
		log.Fatal("incorrect runtime")
	}

	imageClient, conn, err := getImageClient(context.Background(), socketPath)
	logError(err)

	runtimeClient := pb.NewRuntimeServiceClient(conn)

	client := &client{imageClient, runtimeClient}

	config, err := rest.InClusterConfig()
	logError(err)

	clientSet, err := kubernetes.NewForConfig(config)
	logError(err)

	err = writeListImagesToCollectorCR(clientSet, client, socketPath)
	logError(err)

}
