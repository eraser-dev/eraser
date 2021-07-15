package main

import (
	"context"
	"log"
	"os"

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
	Timeout time.Duration
)

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

func getImageClient(ctx context.Context) (pb.ImageServiceClient, *grpc.ClientConn, error) {
	addr, _, err := GetAddressAndDialer("unix:///run/containerd/containerd.sock")
	if err != nil {
		return nil, nil, err
	}

	conn, err := grpc.DialContext(ctx, addr, grpc.WithBlock())

	if err != nil {
		return nil, nil, err
	}

	imageClient := pb.NewImageServiceClient(conn)

	return imageClient, conn, nil
}

func ListImages(ctx context.Context, client pb.ImageServiceClient, image string) (resp *pb.ListImagesResponse, err error) {
	request := &pb.ListImagesRequest{Filter: &pb.ImageFilter{Image: &pb.ImageSpec{Image: image}}}

	resp, err = client.ListImages(ctx, request)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func RemoveImage(ctx context.Context, client pb.ImageServiceClient, image string) (resp *pb.RemoveImageResponse, err error) {
	if image == "" {
		return nil, err
	}

	request := &pb.RemoveImageRequest{Image: &pb.ImageSpec{Image: image}}

	resp, err = client.RemoveImage(ctx, request)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func removeVulnerableImages() (err error) {
	backgroundContext, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	imageClient, conn, err := getImageClient(backgroundContext)

	if err != nil {
		return err
	}

	r, err := ListImages(backgroundContext, imageClient, "")
	if err != nil {
		return err
	}

	var allImages []string
	// map with key: sha id, value: repoTag list (contains full name of image)
	idMap := make(map[string][]string)

	for _, img := range r.Images {
		allImages = append(allImages, img.Id)
		idMap[img.Id] = img.RepoTags
	}

	response, err := pb.NewRuntimeServiceClient(conn).ListContainers(backgroundContext, new(pb.ListContainersRequest))
	if err != nil {
		return err
	}

	var runningImages map[string]struct{}

	for _, container := range response.Containers {
		curr := container.Image
		runningImages[curr.GetImage()] = struct{}{}
	}

	var nonRunningImages map[string]struct{}

	for _, img := range allImages {
		if _, isRunning := runningImages[img]; !isRunning {
			nonRunningImages[img] = struct{}{}
		}
	}

	// TESTING :
	fmt.Println("\nAll images: ")
	fmt.Println(len(allImages))
	for _, img := range allImages {
		fmt.Println(idMap[img], ", ", img)
	}
	fmt.Println("\nNon-running images: ")
	fmt.Println(len(nonRunningImages))
	for key, _ := range nonRunningImages {
		fmt.Println(idMap[key], ", ", key)
	}

	var vulnerableImages []string

	// TODO: change this to read vulnerable images from ImageList
	// adding random image for testing purposes
	vulnerableImages = append(vulnerableImages, "docker.io/ashnam/list_images")

	// remove vulnerable images
	for _, img := range vulnerableImages {
		// image passed in as id
		if _, isNonRunning := nonRunningImages[img]; isNonRunning {
			_, err = RemoveImage(backgroundContext, imageClient, img)
			if err != nil {
				return err
			}
		}
		// image passed in as name
		if idMap[img] != nil {
			if _, isNonRunning := nonRunningImages[idMap[img][0]]; isNonRunning {
				_, err = RemoveImage(backgroundContext, imageClient, idMap[img][0])
				if err != nil {
					return err
				}
			}
		}
	}

	// TESTING :
	fmt.Println("\nAll images following remove: ")
	fmt.Println(len(allImages))
	for _, img := range allImages {
		fmt.Println(idMap[img], ", ", img)
	}

	return nil
}

func main() {

	// TODO: image job should pass the imagelist into each pod as a env variable, and pass that into removeVulnerableImages()
	err := removeVulnerableImages()

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

}
