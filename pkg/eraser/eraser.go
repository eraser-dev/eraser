package main

import (
	"context"
	"flag"
	"log"
	"os"

	"fmt"
	"time"

	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"

	"net"
	"net/url"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
)

const (
	// unixProtocol is the network protocol of unix socket.
	unixProtocol = "unix"
)

var (
	// Timeout  of connecting to server (default: 10s)
	timeout time.Duration = 10 * time.Second
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

func listImages(ctx context.Context, client pb.ImageServiceClient, image string) (resp *pb.ListImagesResponse, err error) {
	request := &pb.ListImagesRequest{Filter: &pb.ImageFilter{Image: &pb.ImageSpec{Image: image}}}

	resp, err = client.ListImages(ctx, request)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func removeImage(ctx context.Context, client pb.ImageServiceClient, image string) (resp *pb.RemoveImageResponse, err error) {
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

func removeVulnerableImages(socketPath string, imagelistName string) (err error) {
	backgroundContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	imageClient, conn, err := getImageClient(backgroundContext, socketPath)

	if err != nil {
		return err
	}

	r, err := listImages(backgroundContext, imageClient, "")
	if err != nil {
		return err
	}

	allImages := make([]string, 0, len(r.Images))
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

	runningImages := make(map[string]struct{})

	for _, container := range response.Containers {
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

	var vulnerableImages []string

	nonRunningNames := make(map[string]struct{}, len(allImages)-len(runningImages))
	remove := ""

	for key := range nonRunningImages {
		if idMap[key] != nil && len(idMap[key]) > 0 {
			nonRunningNames[idMap[key][0]] = struct{}{}
			remove = idMap[key][0]
		}
	}

	// get vulnerable images from ImageList
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	result := eraserv1alpha1.ImageList{}

	// ? v1alpha1.AddToScheme(scheme.Scheme)

	err = clientset.RESTClient().Get().
		AbsPath("../api/v1alpha1").
		Namespace("eraser-system").
		Resource("imagelist").
		Name(imagelistName).
		Do(backgroundContext).Into(&result)

	if err != nil {
		return err
	}

	imagelist := result.Spec.Images

	log.Println("imagelist: ")

	for _, img := range imagelist {
		log.Println(img)
	}

	// TODO: change this to read vulnerable images from ImageList
	// adding random image for testing purposes
	vulnerableImages = append(vulnerableImages, remove)

	// remove vulnerable images
	for _, img := range vulnerableImages {

		// image passed in as id
		if _, isNonRunning := nonRunningImages[img]; isNonRunning {
			_, err = removeImage(backgroundContext, imageClient, img)
			if err != nil {
				return err
			}
		}

		// image passed in as name
		if _, isNonRunning := nonRunningNames[img]; isNonRunning {
			_, err = removeImage(backgroundContext, imageClient, img)
			if err != nil {
				return err
			}
		}

	}

	// TESTING :
	r, err = listImages(backgroundContext, imageClient, "")
	if err != nil {
		return err
	}

	var allImages2 []string

	for _, img := range r.Images {
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

	if *runtimePtr == "dockershim" {
		socketPath = "unix:///var/run/dockershim.sock"
	} else if *runtimePtr == "containerd" {
		socketPath = "unix:///run/containerd/containerd.sock"
	} else if *runtimePtr == "crio" {
		socketPath = "unix:///var/run/crio/crio.sock "
	} else {
		log.Println("incorrect runtime")
		os.Exit(1)
	}

	err := removeVulnerableImages(socketPath, *imageListPtr)

	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
