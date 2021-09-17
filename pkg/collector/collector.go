package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/pkg/util"
)

func logError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getImageResult(imageRepoTag []string, imageRepoDigest []string) (imageResult string) {
	if len(imageRepoTag) == 0 {
		imageResult = imageRepoDigest[0]
	} else {
		imageResult = imageRepoTag[0]
	}
	return imageResult
}

func writeListImagesToCollectorCR(clientSet *kubernetes.Clientset, c util.Client, socketPath string) (err error) {
	backgroundContext, cancel := context.WithTimeout(context.Background(), util.Timeout)
	defer cancel()

	images, err := c.ListImages(backgroundContext)
	logError(err)

	// list of images repo's
	imagesResults := make([]string, 0, len(images))

	// Get imageResults slice from repoTags or repoDigest
	for _, image := range images {
		imagesResults = append(imagesResults, getImageResult(image.RepoTags, image.RepoDigests))
	}

	imageCollectorResult := eraserv1alpha1.ImageCollectorResult{
		TypeMeta: v1.TypeMeta{
			APIVersion: "eraser.sh/v1alpha1",
			Kind:       "ImageCollectorResult",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: "imagecollectorresult-" + os.Getenv("NODE_NAME"),
		},
		Status: eraserv1alpha1.ImageCollectorResultStatus{
			Node:          os.Getenv("NODE_NAME"),
			ImagesResults: imagesResults,
		},
	}

	body, err := json.Marshal(imageCollectorResult)
	logError(err)

	// Create imageCollectorResult object
	_, err = clientSet.RESTClient().Post().
		AbsPath(util.ApiPath).
		Name(imageCollectorResult.Name).
		Resource("imagecollectorresult").
		Body(body).DoRaw(backgroundContext)

	if err != nil {
		log.Println("Could not create imagecollectorresult for node ", os.Getenv("NODE_NAME"))
		return err
	}

	return nil
}

func main() {
	runtimePtr := flag.String("runtime", "docker", "container runtime")
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

	imageClient, conn, err := util.GetImageClient(context.Background(), socketPath)
	logError(err)

	runtimeClient := pb.NewRuntimeServiceClient(conn)

	client := &util.ClientType{imageClient, runtimeClient}

	config, err := rest.InClusterConfig()
	logError(err)

	clientSet, err := kubernetes.NewForConfig(config)
	logError(err)

	err = writeListImagesToCollectorCR(clientSet, client, socketPath)
	logError(err)

}
