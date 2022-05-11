package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/pkg/logger"
	"google.golang.org/grpc"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// unixProtocol is the network protocol of unix socket.
	unixProtocol = "unix"
	apiPath      = "apis/eraser.sh/v1alpha1"
	namespace    = "default"
)

var (
	// Timeout  of connecting to server (default: 5m).
	timeout                  = 5 * time.Minute
	errProtocolNotSupported  = errors.New("protocol not supported")
	errEndpointDeprecated    = errors.New("endpoint is deprecated, please consider using full url format")
	errOnlySupportUnixSocket = errors.New("only support unix socket endpoint")
	log                      = logf.Log.WithName("collector")
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

func GetAddressAndDialer(endpoint string) (string, func(ctx context.Context, addr string) (net.Conn, error), error) {
	protocol, addr, err := parseEndpointWithFallbackProtocol(endpoint, unixProtocol)
	if err != nil {
		return "", nil, err
	}
	if protocol != unixProtocol {
		return "", nil, errOnlySupportUnixSocket
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
		return "", "", fmt.Errorf("using %q as %w", endpoint, errEndpointDeprecated)

	default:
		return u.Scheme, "", fmt.Errorf("%q: %w", u.Scheme, errProtocolNotSupported)
	}
}

func getAllImages(c Client) ([]eraserv1alpha1.Image, error) {
	backgroundContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	images, err := c.listImages(backgroundContext)
	if err != nil {
		return nil, err
	}

	allImages := make([]eraserv1alpha1.Image, 0, len(images))

	for _, img := range images {
		currImage := eraserv1alpha1.Image{
			Digest: img.Id,
		}
		if len(img.RepoTags) > 0 {
			currImage.Name = img.RepoTags[0]
		}

		allImages = append(allImages, currImage)

	}

	return allImages, nil
}

func createCollectorCR(ctx context.Context, allImages []eraserv1alpha1.Image) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Info("Could not create InClusterConfig")
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Info("Could not create clientset")
		return err
	}

	imageCollector := eraserv1alpha1.ImageCollector{
		TypeMeta: v1.TypeMeta{
			APIVersion: "eraser.sh/v1alpha1",
			Kind:       "ImageCollector",
		},
		ObjectMeta: v1.ObjectMeta{
			// imagejob will set node name as env when creating collector pod
			Name:      "imagecollector-" + os.Getenv("NODE_NAME"),
			Namespace: namespace,
		},
		Spec: eraserv1alpha1.ImageCollectorSpec{
			Images: allImages,
		},
		// filling in status for testing purposes
		Status: eraserv1alpha1.ImageCollectorStatus{
			Result: allImages,
		},
	}

	body, err := json.Marshal(imageCollector)
	if err != nil {
		log.Info("Could not marshal imagecollector for node: ", os.Getenv("NODE_NAME"))
		return err
	}

	_, err = clientset.RESTClient().Post().
		AbsPath(apiPath).
		Namespace(namespace).
		Name(imageCollector.Name).
		Resource("imagecollectors").
		Body(body).DoRaw(ctx)

	if err != nil {
		log.Error(err, "ERROR: Could not create imagecollector", imageCollector)
		return err
	}

	return nil
}

func main() {
	runtimePtr := flag.String("runtime", "containerd", "container runtime")

	flag.Parse()

	if err := logger.Configure(); err != nil {
		fmt.Fprintln(os.Stderr, "Error setting up logger:", err)
		os.Exit(1)
	}

	var socketPath string

	switch runtime := *runtimePtr; runtime {
	case "docker":
		socketPath = "unix:///var/run/dockershim.sock"
	case "containerd":
		socketPath = "unix:///run/containerd/containerd.sock"
	case "cri-o":
		socketPath = "unix:///var/run/crio/crio.sock"
	default:
		log.Error(fmt.Errorf("unsupported runtime"), "runtime", runtime)
		os.Exit(1)
	}

	imageclient, conn, err := getImageClient(context.Background(), socketPath)
	if err != nil {
		log.Error(err, "failed to get image client")
		os.Exit(1)
	}

	runTimeClient := pb.NewRuntimeServiceClient(conn)

	client := &client{imageclient, runTimeClient}

	allImages, err := getAllImages(client)

	if err != nil {
		log.Error(err, "failed to list all images")
		os.Exit(1)
	}

	createCollectorCR(context.Background(), allImages)
}
