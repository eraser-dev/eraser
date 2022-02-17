package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"time"

	"google.golang.org/grpc"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
)

const (
	// unixProtocol is the network protocol of unix socket.
	unixProtocol = "unix"
	apiPath      = "apis/eraser.sh/v1alpha1"
)

var (
	// Timeout  of connecting to server (default: 10s).
	timeout                  = 10 * time.Second
	errProtocolNotSupported  = errors.New("protocol not supported")
	errEndpointDeprecated    = errors.New("endpoint is deprecated, please consider using full url format")
	errOnlySupportUnixSocket = errors.New("only support unix socket endpoint")
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

func updateStatus(ctx context.Context, clientset *kubernetes.Clientset, results []eraserv1alpha1.NodeCleanUpDetail) error {
	imageStatus := eraserv1alpha1.ImageStatus{
		TypeMeta: v1.TypeMeta{
			APIVersion: "eraser.sh/v1alpha1",
			Kind:       "ImageStatus",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: "imagestatus-" + os.Getenv("NODE_NAME"),
		},
		Result: eraserv1alpha1.NodeCleanUpResult{
			Node:    os.Getenv("NODE_NAME"),
			Results: results,
		},
	}

	body, err := json.Marshal(imageStatus)
	if err != nil {
		return err
	}

	// create imageStatus object
	_, err = clientset.RESTClient().Post().
		AbsPath(apiPath).
		Name(imageStatus.Name).
		Resource("imagestatuses").
		Body(body).DoRaw(ctx)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			result := eraserv1alpha1.ImageStatus{}
			if err = clientset.RESTClient().Get().
				AbsPath(apiPath).
				Resource("imagestatuses").
				Name(imageStatus.Name).
				Do(context.Background()).Into(&result); err != nil {
				log.Println("Could not get imagestatus", imageStatus.Name)
				return err
			}

			result.Result.Results = imageStatus.Result.Results
			body, err := json.Marshal(result)
			if err != nil {
				return err
			}
			_, err = clientset.RESTClient().Put().
				AbsPath(apiPath).
				Name(imageStatus.Name).
				Resource("imagestatuses").
				Body(body).DoRaw(ctx)
			if err != nil {
				log.Println("Could not update imagestatus for node: ", os.Getenv("NODE_NAME"))
				return err
			}
		}
		log.Println("Could not create imagestatus for  node: ", os.Getenv("NODE_NAME"))
		return err
	}

	return nil
}

func mapContainsValue(idMap map[string][]string, img string) bool {
	for _, v := range idMap {
		if len(v) > 0 {
			if v[0] == img {
				return true
			}
		}
	}
	return false
}

func removeImages(clientset *kubernetes.Clientset, c Client, socketPath string, targetImages []string) error {
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

	// holds ids of running images
	runningImages := make(map[string]struct{}, len(containers))
	for _, container := range containers {
		curr := container.Image
		runningImages[curr.GetImage()] = struct{}{}
	}

	// map for non-running images by id
	nonRunningImages := make(map[string]struct{}, len(allImages)-len(runningImages))
	for _, img := range allImages {
		if _, isRunning := runningImages[img]; !isRunning {
			nonRunningImages[img] = struct{}{}
		}
	}

	// map for non-running imags by name
	nonRunningNames := make(map[string]struct{}, len(allImages)-len(runningImages))
	for key := range nonRunningImages {
		if idMap[key] != nil && len(idMap[key]) > 0 {
			nonRunningNames[idMap[key][0]] = struct{}{}
		}
	}

	var results []eraserv1alpha1.NodeCleanUpDetail

	// remove target images
	for _, img := range targetImages {
		_, isNonRunningNames := nonRunningNames[img]
		_, isNonRunningImages := nonRunningImages[img]

		if isNonRunningImages || isNonRunningNames {
			err = c.deleteImage(backgroundContext, img)
			log.Println("Deleting img: ", img)
			if err != nil {
				results = append(results, eraserv1alpha1.NodeCleanUpDetail{
					ImageName: img,
					Status:    eraserv1alpha1.Error,
					Message:   err.Error(),
				})
			} else {
				results = append(results, eraserv1alpha1.NodeCleanUpDetail{
					ImageName: img,
					Status:    eraserv1alpha1.Success,
					Message:   "successfully removed image",
				})
			}
		} else {
			isRunningName := mapContainsValue(idMap, img)
			_, isRunningID := runningImages[img]
			if isRunningName || isRunningID {
				results = append(results, eraserv1alpha1.NodeCleanUpDetail{
					ImageName: img,
					Status:    eraserv1alpha1.Error,
					Message:   "image is running",
				})
			} else {
				results = append(results, eraserv1alpha1.NodeCleanUpDetail{
					ImageName: img,
					Status:    eraserv1alpha1.Error,
					Message:   "image not found",
				})
			}
		}
	}

	if err := updateStatus(backgroundContext, clientset, results); err != nil {
		return err
	}

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

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	result := eraserv1alpha1.ImageList{}
	if err = clientset.RESTClient().Get().
		AbsPath(apiPath).
		Resource("imagelists").
		Name(*imageListPtr).
		Do(context.Background()).Into(&result); err != nil {
		log.Println("Unable to find imagelist", " Name: "+*imageListPtr, " AbsPath: ", apiPath)
		log.Fatal(err)
	}

	if err := removeImages(clientset, client, socketPath, result.Spec.Images); err != nil {
		log.Fatal(err)
	}
}
