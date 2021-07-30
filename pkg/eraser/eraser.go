package main

import (
	"context"
	"flag"
	"log"

	"fmt"
	"time"

	"google.golang.org/grpc"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"

	"net"
	"net/url"
)

const (
	// unixProtocol is the network protocol of unix socket.
	unixProtocol = "unix"
)

var (
	// Timeout  of connecting to server (default: 10s)
	timeout = 10 * time.Second
)

type client struct {
	images  pb.ImageServiceClient
	runtime pb.RuntimeServiceClient
}

type Client interface {
	listImages(context.Context) ([]*pb.Image, error)
	listContainers(context.Context) ([]*pb.Container, error)
	deleteImage(context.Context, string) error
}

func (c *client) listContainers(context.Context) (list []*pb.Container, err error) {
	resp, err := c.runtime.ListContainers(context.Background(), new(pb.ListContainersRequest))
	if err != nil {
		return nil, err
	}
	return resp.Containers, nil
}

func (c *client) listImages(ctx context.Context) (list []*pb.Image, err error) {
	request := &pb.ListImagesRequest{Filter: nil}

	resp, err := c.images.ListImages(ctx, request)
	if err != nil {
		return nil, err
	}

	return resp.Images, nil
}

func (c *client) deleteImage(ctx context.Context, image string) (err error) {
	if image == "" {
		return err
	}

	request := &pb.RemoveImageRequest{Image: &pb.ImageSpec{Image: image}}

	_, err = c.images.RemoveImage(ctx, request)
	if err != nil {
		return err
	}

	return nil
}

func GetAddressAndDialer(endpoint string) (string, func(ctx context.Context, addr string) (net.Conn, error), error) {
	protocol, addr, err := parseEndpointWithFallbackProtocol(endpoint, unixProtocol)
	if err != nil {
		return "", nil, err
	}
	if protocol != unixProtocol {
		return "", nil, fmt.Errorf("only support unix socket endpoint")
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
		return "", "", err
	}

	switch u.Scheme {
	case "tcp":
		return "tcp", u.Host, nil

	case "unix":
		return "unix", u.Path, nil

	case "":
		return "", "", fmt.Errorf("using %q as endpoint is deprecated, please consider using full url format", endpoint)

	default:
		return u.Scheme, "", fmt.Errorf("protocol %q not supported", u.Scheme)
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

func removeImages(c Client, socketPath string, imagelistName string) (err error) {
	backgroundContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	images, err := c.listImages(backgroundContext)
	if err != nil {
		return err
	}

	allImages := make([]string, 0, len(images))
	// map with key: sha id, value: repoTag list (contains full name of image)
	idMap := make(map[string][]string)

	for _, img := range images {
		allImages = append(allImages, img.Id)
		idMap[img.Id] = img.RepoTags
	}

	containers, err := c.listContainers(backgroundContext)
	if err != nil {
		return err
	}

	runningImages := make(map[string]struct{}, len(containers))

	for _, container := range containers {
		curr := container.Image
		runningImages[curr.GetImage()] = struct{}{}
	}

	nonRunningImages := make(map[string]struct{}, len(allImages)-len(runningImages))

	for _, img := range allImages {
		if _, isRunning := runningImages[img]; !isRunning {
			nonRunningImages[img] = struct{}{}
		}
	}

	// TESTING :
	log.Println("\nAll images: ")
	log.Println(len(allImages))

	var targetImages []string

	nonRunningNames := make(map[string]struct{}, len(allImages)-len(runningImages))
	remove := ""

	for key := range nonRunningImages {
		if idMap[key] != nil && len(idMap[key]) > 0 {
			nonRunningNames[idMap[key][0]] = struct{}{}
			remove = idMap[key][0]
		}
	}

	// TODO: change this to read target images from ImageList
	// adding random image for testing purposes
	targetImages = append(targetImages, remove)

	// remove target images
	for _, img := range targetImages {
		// image passed in as id
		if _, isNonRunning := nonRunningImages[img]; isNonRunning {
			err = c.deleteImage(backgroundContext, img)
			if err != nil {
				return err
			}
		}

		// image passed in as name
		if _, isNonRunning := nonRunningNames[img]; isNonRunning {
			err = c.deleteImage(backgroundContext, img)
			if err != nil {
				return err
			}
		}

	}

	// TESTING :
	imageTest, err := c.listImages(backgroundContext)
	if err != nil {
		return err
	}

	allImages2 := make([]string, 0, len(allImages))

	for _, img := range imageTest {
		allImages2 = append(allImages2, img.Id)
	}

	log.Println("\nAll images following remove: ")
	log.Println(len(allImages2))

	return nil
}

func main() {
	runtimePtr := flag.String("runtime", "containerd", "container runtime")
	imageListPtr := flag.String("imagelist", "", "name of ImageList")

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

	imageclient, conn, err := getImageClient(context.Background(), socketPath)
	if err != nil {
		log.Fatal(err)
	}

	runTimeClient := pb.NewRuntimeServiceClient(conn)

	client := &client{imageclient, runTimeClient}

	err = removeImages(client, socketPath, *imageListPtr)

	if err != nil {
		log.Fatal(err)
	}
}
