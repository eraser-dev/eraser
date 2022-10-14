package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
)

const (
	// unixProtocol is the network protocol of unix socket.
	unixProtocol             = "unix"
	PipeMode                 = 0o644
	ScanErasePath            = "/run/eraser.sh/shared-data/scanErase"
	CollectScanPath          = "/run/eraser.sh/shared-data/collectScan"
	EraseCompleteCollectPath = "/run/eraser.sh/shared-data/eraseCompleteCollect"
	EraseCompleteMessage     = "complete"
	EraseCompleteScanPath    = "/run/eraser.sh/shared-data/eraseCompleteScan"
)

type ExclusionList struct {
	Excluded []string `json:"excluded"`
}

var (
	ErrProtocolNotSupported  = errors.New("protocol not supported")
	ErrEndpointDeprecated    = errors.New("endpoint is deprecated, please consider using full url format")
	ErrOnlySupportUnixSocket = errors.New("only support unix socket endpoint")

	RuntimeSocketPathMap = map[string]string{
		"docker":     "unix:///var/run/dockershim.sock",
		"containerd": "unix:///run/containerd/containerd.sock",
		"cri-o":      "unix:///var/run/crio/crio.sock",
	}
)

func GetAddressAndDialer(endpoint string) (string, func(ctx context.Context, addr string) (net.Conn, error), error) {
	protocol, addr, err := ParseEndpointWithFallbackProtocol(endpoint, unixProtocol)
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

func ParseEndpointWithFallbackProtocol(endpoint string, fallbackProtocol string) (protocol string, addr string, err error) {
	if protocol, addr, err = ParseEndpoint(endpoint); err != nil && protocol == "" {
		fallbackEndpoint := fallbackProtocol + "://" + endpoint
		protocol, addr, err = ParseEndpoint(fallbackEndpoint)
		if err != nil {
			return "", "", err
		}
	}
	return protocol, addr, err
}

func ParseEndpoint(endpoint string) (string, string, error) {
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

func GetImageClient(ctx context.Context, socketPath string) (pb.ImageServiceClient, *grpc.ClientConn, error) {
	addr, dialer, err := GetAddressAndDialer(socketPath)
	if err != nil {
		return nil, nil, err
	}

	conn, err := grpc.DialContext(ctx, addr, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(dialer))
	if err != nil {
		return nil, nil, err
	}

	imageClient := pb.NewImageServiceClient(conn)

	return imageClient, conn, nil
}

func ListImages(ctx context.Context, images pb.ImageServiceClient) (list []*pb.Image, err error) {
	request := &pb.ListImagesRequest{Filter: nil}

	resp, err := images.ListImages(ctx, request)
	if err != nil {
		return nil, err
	}

	return resp.Images, nil
}

func ListContainers(ctx context.Context, runtime pb.RuntimeServiceClient) (list []*pb.Container, err error) {
	resp, err := runtime.ListContainers(context.Background(), new(pb.ListContainersRequest))
	if err != nil {
		return nil, err
	}
	return resp.Containers, nil
}

func GetRunningImages(containers []*pb.Container, idToTagListMap map[string][]string) map[string]string {
	// Images that are running
	// map of (digest | tag) -> digest
	runningImages := make(map[string]string)
	for _, container := range containers {
		curr := container.Image
		digest := curr.GetImage()
		runningImages[digest] = digest

		for _, tag := range idToTagListMap[digest] {
			runningImages[tag] = digest
		}
	}
	return runningImages
}

func GetNonRunningImages(runningImages map[string]string, allImages []string, idToTagListMap map[string][]string) map[string]string {
	// Images that aren't running
	// map of (digest | tag) -> digest
	nonRunningImages := make(map[string]string)

	for _, digest := range allImages {
		if _, isRunning := runningImages[digest]; !isRunning {
			nonRunningImages[digest] = digest

			for _, tag := range idToTagListMap[digest] {
				nonRunningImages[tag] = digest
			}
		}
	}

	return nonRunningImages
}

func IsExcluded(excluded map[string]struct{}, img string, idToTagListMap map[string][]string) bool {
	if len(excluded) == 0 {
		return false
	}

	// check if img excluded by digest
	if _, contains := excluded[img]; contains {
		return true
	}

	// check if img excluded by name
	for _, imgName := range idToTagListMap[img] {
		if _, contains := excluded[imgName]; contains {
			return true
		}
	}

	// look for excluded repository values and names without tag
	for key := range excluded {
		// if excluded key ends in /*, check image with pattern match
		if strings.HasSuffix(key, "/*") {
			// store repository name
			repo := strings.Split(key, "*")

			// check if img is part of repo
			if match := strings.HasPrefix(img, repo[0]); match {
				return true
			}

			// retrieve and check by name in the case img is digest
			for _, imgName := range idToTagListMap[img] {
				if match := strings.HasPrefix(imgName, repo[0]); match {
					return true
				}
			}
		}

		// if excluded key ends in :*, check image with pattern patch
		if strings.HasSuffix(key, ":*") {
			// store image name
			imagePath := strings.Split(key, ":")

			if match := strings.HasPrefix(img, imagePath[0]); match {
				return true
			}

			// retrieve and check by name in the case img is digest
			for _, imgName := range idToTagListMap[img] {
				if match := strings.HasPrefix(imgName, imagePath[0]); match {
					return true
				}
			}
		}
	}

	return false
}

func ParseImageList(path string) ([]string, error) {
	imagelist := []string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &imagelist); err != nil {
		return nil, err
	}

	return imagelist, nil
}

func ParseExcluded() (map[string]struct{}, error) {
	excludedMap := make(map[string]struct{})
	var excludedList []string

	files, err := os.ReadDir("./")
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "exclude-") {
			temp, err := readConfigMap(file.Name())
			if err != nil {
				return nil, err
			}
			excludedList = append(excludedList, temp...)
		}
	}

	for _, img := range excludedList {
		excludedMap[img] = struct{}{}
	}

	return excludedMap, nil
}

func BoolPtr(b bool) *bool {
	return &b
}

func readConfigMap(path string) ([]string, error) {
	var fileName string

	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			fileName = f.Name()
			break
		}
	}

	var images []string
	data, err := os.ReadFile(path + "/" + fileName)

	if os.IsNotExist(err) {
		return nil, err
	} else if err != nil {
		return nil, err
	}

	var result ExclusionList
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	images = append(images, result.Excluded...)

	return images, nil
}

func ReadCollectScanPipe(ctx context.Context) ([]eraserv1alpha1.Image, error) {
	timer := time.NewTimer(time.Second)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()

	var f *os.File
	for {
		var err error

		f, err = os.OpenFile(CollectScanPath, os.O_RDONLY, 0)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return nil, err
		}

		timer.Reset(time.Second)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			continue
		}
	}

	// json data is list of []eraserv1alpha1.Image
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	allImages := []eraserv1alpha1.Image{}
	if err = json.Unmarshal(data, &allImages); err != nil {
		return nil, err
	}

	return allImages, nil
}

func WriteScanErasePipe(vulnerableImages []eraserv1alpha1.Image) error {
	data, err := json.Marshal(vulnerableImages)
	if err != nil {
		return err
	}

	if err = unix.Mkfifo(ScanErasePath, PipeMode); err != nil {
		return err
	}

	file, err := os.OpenFile(ScanErasePath, os.O_WRONLY, 0)
	if err != nil {
		return err
	}

	if _, err := file.Write(data); err != nil {
		return err
	}

	return file.Close()
}
