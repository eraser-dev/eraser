package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/pkg/logger"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	util "github.com/Azure/eraser/pkg/utils"
)

const (
	apiPath = "apis/eraser.sh/v1alpha1"
)

var (
	// Timeout  of connecting to server (default: 5m).
	timeout = 5 * time.Minute
	log     = logf.Log.WithName("collector")
)

type client struct {
	images  pb.ImageServiceClient
	runtime pb.RuntimeServiceClient
}

type Client interface {
	listImages(context.Context) ([]*pb.Image, error)
}

func (c *client) listImages(ctx context.Context) (list []*pb.Image, err error) {
	return util.ListImages(ctx, c.images)
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
			GenerateName: "imagecollector-" + os.Getenv("NODE_NAME") + "-",
		},
		Spec: eraserv1alpha1.ImageCollectorSpec{
			Images: allImages,
		},
	}

	body, err := json.Marshal(imageCollector)
	if err != nil {
		log.Info("Could not marshal imagecollector for node: ", os.Getenv("NODE_NAME"))
		return err
	}

	_, err = clientset.RESTClient().Post().
		AbsPath(apiPath).
		Resource("imagecollectors").
		Body(body).DoRaw(ctx)

	if err != nil {
		log.Error(err, "Could not create imagecollector", imageCollector.Name, imageCollector.APIVersion)
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

	imageclient, conn, err := util.GetImageClient(context.Background(), socketPath)
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

	if err := createCollectorCR(context.Background(), allImages); err != nil {
		log.Error(err, "Error creating ImageCollector CR")
		os.Exit(1)
	}
}
