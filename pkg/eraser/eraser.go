package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"

	"fmt"
	"time"

	"google.golang.org/grpc"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	apiPath      = "apis/eraser.sh/v1alpha1"
	namespace    = "eraser-system"
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

func updateStatus(results []eraserv1alpha1.NodeCleanUpDetail) {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	imageStatus := eraserv1alpha1.ImageStatus{
		TypeMeta: v1.TypeMeta{
			APIVersion: "eraser.sh/v1alpha1",
			Kind:       "ImageStatus",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "imagestatus-" + os.Getenv("NODE_NAME"),
			Namespace: "eraser-system",
		},
		Result: eraserv1alpha1.NodeCleanUpResult{
			Node:    os.Getenv("NODE_NAME"),
			Results: results,
		},
	}

	body, err := json.Marshal(imageStatus)
	if err != nil {
		log.Println(err)
	}

	// create imageStatus object
	res, err := clientset.RESTClient().Post().
		AbsPath("apis/eraser.sh/v1alpha1").
		Namespace("eraser-system").
		Name(imageStatus.Name).
		Resource("imagestatuses").
		Body(body).DoRaw(context.TODO())

	// verify object output
	log.Print(string(res))

	if err != nil {
		log.Println(err)
		log.Println("Could not create imagestatus for  node: ", os.Getenv("NODE_NAME"))
	}
}

func removeImages(c Client, socketPath string, targetImages []string) (err error) {
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

	nonRunningNames := make(map[string]struct{}, len(allImages)-len(runningImages))
	remove := ""

	for key := range nonRunningImages {
		if idMap[key] != nil && len(idMap[key]) > 0 {
			nonRunningNames[idMap[key][0]] = struct{}{}
			// delete later, for testing
			remove = idMap[key][0]
		}
	}

	// add an image to remove for testing
	targetImages = append(targetImages, remove)

	log.Println("\n\nTarget images: (1 additional added to ImageList to test remove)")
	for _, img := range targetImages {
		log.Println(img)
	}

	var results []eraserv1alpha1.NodeCleanUpDetail

	// remove target images
	for _, img := range targetImages {
		_, isNonRunningNames := nonRunningNames[img]
		_, isNonRunningImages := nonRunningImages[img]

		if isNonRunningImages || isNonRunningNames {
			err = c.deleteImage(backgroundContext, img)
			if err != nil {
				results = append(results, eraserv1alpha1.NodeCleanUpDetail{
					ImageName: img,
					Status:    "error",
					Message:   err.Error(),
				})
			} else {
				results = append(results, eraserv1alpha1.NodeCleanUpDetail{
					ImageName: img,
					Status:    "success",
					Message:   "successfully removed image",
				})
			}
		} else {
			if _, isRunning := runningImages[img]; isRunning {
				results = append(results, eraserv1alpha1.NodeCleanUpDetail{
					ImageName: img,
					Status:    "error",
					Message:   "image is running",
				})
			} else {
				results = append(results, eraserv1alpha1.NodeCleanUpDetail{
					ImageName: img,
					Status:    "error",
					Message:   "image not found",
				})
			}
		}
	}

	updateStatus(results)

	// TESTING :
	imageTest, err := c.listImages(backgroundContext)
	if err != nil {
		return err
	}

	allImages2 := make([]string, 0, len(allImages))

	for _, img := range imageTest {
		allImages2 = append(allImages2, img.Id)
	}

	log.Println("\n\nAll images following remove: ")
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

	// get list of images to remove from ImageList
	var targetImages []string
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	result := eraserv1alpha1.ImageList{}
	err = clientset.RESTClient().Get().
		AbsPath(apiPath).
		Namespace(namespace).
		Resource("imagelists").
		Name(*imageListPtr).
		Do(context.Background()).Into(&result)

	if err != nil {
		log.Println("Unable to find imagelist", " Name: "+*imageListPtr, " AbsPath: ", apiPath)
		log.Fatal(err)
	}

	// set target images to imagelist values
	targetImages = result.Spec.Images

	err = removeImages(client, socketPath, targetImages)

	if err != nil {
		log.Fatal(err)
	}
}
