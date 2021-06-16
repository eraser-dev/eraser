package main

import (
	"context"
	//	"flag"
	"fmt"
	"time"

	//	"github.com/containerd/containerd"
	//	"github.com/docker/docker/api/types"
	//	"github.com/docker/docker/client"

	cli "github.com/urfave/cli/v2"
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
	Timeout time.Duration
)

/*

func getContainerdImages() error {

	client, err := containerd.New("/run/containerd/containerd.sock", containerd.WithDefaultNamespace("k8s.io"))
	if err != nil {
		return err
	}
	defer client.Close()

	ctx := context.Background()

	list, err := client.ListImages(ctx)
	if err != nil {
		return err
	}

	for _, elm := range list {
		fmt.Println(elm.Name())
	}

	return nil
}

func getDockerImages() error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	images, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return err
	}

	for _, image := range images {
		fmt.Println(image.RepoTags)
	}

	return nil

}

*/

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
		if err == nil {
			//klog.InfoS("Using this endpoint is deprecated, please consider using full URL format", "endpoint", endpoint, "URL", fallbackEndpoint)
		}
	}
	return
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

func getImageClient(context *cli.Context) (pb.ImageServiceClient, *grpc.ClientConn, error) {
	addr, dialer, err := GetAddressAndDialer("unix:///run/containerd/containerd.sock")

	if err != nil {
		fmt.Print("get address and dialer err")
	}

	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(Timeout), grpc.WithContextDialer(dialer))

	if err != nil {
		return nil, nil, err
	}

	imageClient := pb.NewImageServiceClient(conn)

	return imageClient, conn, nil
}

func ListImages(client pb.ImageServiceClient, image string) (resp *pb.ListImagesResponse, err error) {
	request := &pb.ListImagesRequest{Filter: &pb.ImageFilter{Image: &pb.ImageSpec{Image: image}}}
	//logrus.Debugf("ListImagesRequest: %v", request)

	resp, err = client.ListImages(context.Background(), request)
	//logrus.Debugf("ListImagesResponse: %v", resp)

	return
}

func RemoveImage(client pb.ImageServiceClient, image string) (resp *pb.RemoveImageResponse, err error) {
	if image == "" {
		return nil, fmt.Errorf("ImageID cannot be empty")
	}
	request := &pb.RemoveImageRequest{Image: &pb.ImageSpec{Image: image}}
	//logrus.Debugf("RemoveImageRequest: %v", request)

	resp, err = client.RemoveImage(context.Background(), request)
	//logrus.Debugf("RemoveImageResponse: %v", resp)
	return
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func main() {

	/*
		runtimePtr := flag.String("runtime", "containerd", "container runtime")

		flag.Parse()

		if *runtimePtr == "docker" {
			getDockerImages()
		}

		if *runtimePtr == "containerd" {
			getContainerdImages()
		}
	*/

	// using CRICTL

	ctx := cli.NewContext(nil, nil, nil)

	imageClient, conn, err := getImageClient(ctx)

	if err != nil {
		fmt.Printf("image client err")
	}

	r, err := ListImages(imageClient, "")
	if err != nil {
		fmt.Printf("list err")
	}

	var allImages []string

	for _, img := range r.Images {
		allImages = append(allImages, img.Id)
	}

	response, err := pb.NewRuntimeServiceClient(conn).ListContainers(context.Background(), new(pb.ListContainersRequest))

	if err != nil {
		fmt.Printf("list containers err")
	}

	var runningImages []string

	for _, container := range response.Containers {
		runningImages = append(runningImages, container.ImageRef)
	}

	var nonRunningImages []string

	for _, img := range allImages {
		if !contains(runningImages, img) {
			nonRunningImages = append(nonRunningImages, img)
		}
	}

	// quick remove test

	fmt.Println("All images: ")
	fmt.Println(len(allImages))

	fmt.Println("Running images: ")
	fmt.Println(len(runningImages))

	fmt.Println("non running images: ")
	fmt.Println(len(nonRunningImages))

	fmt.Println("Removing first non-running image ...")
	RemoveImage(imageClient, nonRunningImages[0])

	r, err = ListImages(imageClient, "")

	fmt.Println("New All images total: ")
	fmt.Println(len(r.Images))

}
