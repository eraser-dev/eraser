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

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Azure/eraser/pkg/logger"
)

const (
	// unixProtocol is the network protocol of unix socket.
	unixProtocol = "unix"
)

var (
	// Timeout  of connecting to server (default: 5m).
	timeout                  = 5 * time.Minute
	errProtocolNotSupported  = errors.New("protocol not supported")
	errEndpointDeprecated    = errors.New("endpoint is deprecated, please consider using full url format")
	errOnlySupportUnixSocket = errors.New("only support unix socket endpoint")
	log                      = logf.Log.WithName("eraser")
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
		if status.Code(err) == codes.NotFound {
			return nil
		}
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

func removeImages(c Client, targetImages []string) error {
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
	nonRunningImages := make(map[string]string)
	for _, img := range allImages {
		if _, isRunning := runningImages[img]; !isRunning {
			nonRunningImages[img] = img
		}
	}

	// Debug logs
	log.V(1).Info("List of non-running images by digest", "nonRunningImages", nonRunningImages)
	log.V(1).Info("Map of digest to image name(s)", "idMap", idMap)

	// add names to map, to resolve them to digests
	for digest := range nonRunningImages {
		if idMap[digest] != nil && len(idMap[digest]) > 0 {
			for _, name := range idMap[digest] {
				nonRunningImages[name] = digest
			}
		}
	}

	// remove target images
	var prune bool
	deletedImages := make(map[string]struct{}, len(targetImages))
	for _, img := range targetImages {
		if img == "*" {
			prune = true
			continue
		}

		digest, isNonRunning := nonRunningImages[img]
		if isNonRunning {
			err = c.deleteImage(backgroundContext, digest)
			if err != nil {
				log.Error(err, "Error removing", "image", digest)
			} else {
				deletedImages[img] = struct{}{}
				log.Info("Removed", "given", img, "digest", digest, "digest", digest)
			}
		} else {
			isRunningName := mapContainsValue(idMap, img)
			_, isRunningID := runningImages[img]
			if isRunningName || isRunningID {
				log.Info("Image is running", "image", img)
			} else {
				log.Info("Image is not on node", "image", img)
			}
		}
	}

	if prune {
		for img := range nonRunningImages {
			_, deleted := deletedImages[img]
			if deleted {
				continue
			}

			_, running := runningImages[img]
			if running {
				continue
			}

			if err := c.deleteImage(backgroundContext, img); err != nil {
				log.Error(err, "Error during prune", "image", img)
				continue
			}
			log.Info("Prune successful", "image", img)
		}
	}

	return nil
}

func main() {
	runtimePtr := flag.String("runtime", "containerd", "container runtime")
	imageListPtr := flag.String("imagelist", "", "name of ImageList")

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

	data, err := os.ReadFile(*imageListPtr)
	if err != nil {
		log.Error(err, "failed to read image list file")
		os.Exit(1)
	}

	var ls []string
	if err := json.Unmarshal(data, &ls); err != nil {
		log.Error(err, "failed to unmarshal image list")
		os.Exit(1)
	}

	if err := removeImages(client, ls); err != nil {
		log.Error(err, "failed to remove images")
		os.Exit(1)
	}
}
