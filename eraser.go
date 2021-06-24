package main

import (
	"context"
	"io/ioutil"
	"log"

	"fmt"
	"time"

	cli "github.com/urfave/cli/v2"
	"github.com/utahta/go-openuri"
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

	resp, err = client.ListImages(context.Background(), request)

	return
}

func RemoveImage(client pb.ImageServiceClient, image string) (resp *pb.RemoveImageResponse, err error) {
	if image == "" {
		return nil, fmt.Errorf("ImageID cannot be empty")
	}
	request := &pb.RemoveImageRequest{Image: &pb.ImageSpec{Image: image}}

	resp, err = client.RemoveImage(context.Background(), request)

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
	m := make(map[string][]string)

	for _, img := range r.Images {
		allImages = append(allImages, img.Id)
		m[img.Id] = img.RepoTags
	}

	response, err := pb.NewRuntimeServiceClient(conn).ListContainers(context.Background(), new(pb.ListContainersRequest))

	if err != nil {
		fmt.Printf("list containers err")
	}

	var runningImages []string

	for _, container := range response.Containers {
		curr := container.Image
		runningImages = append(runningImages, curr.GetImage())
	}

	var nonRunningImages []string

	for _, img := range allImages {
		if !contains(runningImages, img) {
			nonRunningImages = append(nonRunningImages, img)
		}
	}

	// testing

	fmt.Println("\nAll images: ")
	fmt.Println(len(allImages))
	for _, img := range allImages {
		fmt.Println(m[img], ", ", img)
	}

	fmt.Println("\nRunning images: ")
	fmt.Println(len(runningImages))
	for _, img := range runningImages {
		fmt.Println(m[img], ", ", img)
	}

	fmt.Println("\nNon-running images: ")
	fmt.Println(len(nonRunningImages))
	for _, img := range nonRunningImages {
		fmt.Println(m[img], ", ", img)
	}

	o, err := openuri.Open("https://gist.githubusercontent.com/ashnamehrotra/1a244c8fae055bce853fd344ac4c5e02/raw/98baf0a4f0864b3dcc48523a9bddd28938fecd17/vulnerable.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer o.Close()

	b, _ := ioutil.ReadAll(o)
	fmt.Println("Read vulnerable: ")
	fmt.Println(string(b))

	var vulnerableImages []string

	// add a vulnerable image to test
	vulnerableImages = append(vulnerableImages, nonRunningImages[0])

	fmt.Println("\nVulnerable images: ")
	fmt.Println(len(vulnerableImages))
	for _, img := range vulnerableImages {
		fmt.Println(img)
	}

	fmt.Println("\nRemoving vulnerable images ...")
	for _, img := range vulnerableImages {
		if contains(nonRunningImages, img) {
			RemoveImage(imageClient, img)
		}
	}

	fmt.Println("All images following remove:")

	r, err = ListImages(imageClient, "")

	if err != nil {
		fmt.Printf("list img err")
	}

	fmt.Println(len(r.Images))
	for _, img := range r.Images {
		fmt.Println(m[img.Id], ", ", img.Id)
	}

}
