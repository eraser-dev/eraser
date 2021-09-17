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
		AbsPath(util.ApiPath).
		Name(imageStatus.Name).
		Resource("imagestatuses").
		Body(body).DoRaw(ctx)

	if err != nil {
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

func removeImages(clientset *kubernetes.Clientset, c util.Client, socketPath string, targetImages []string) (err error) {
	backgroundContext, cancel := context.WithTimeout(context.Background(), util.Timeout)
	defer cancel()

	images, err := c.ListImages(backgroundContext)
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

	containers, err := c.ListContainers(backgroundContext)
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
			err = c.DeleteImage(backgroundContext, img)
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
			_, isRunningId := runningImages[img]
			if isRunningName || isRunningId {
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

	updateStatus(backgroundContext, clientset, results)

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

	imageclient, conn, err := util.GetImageClient(context.Background(), socketPath)
	if err != nil {
		log.Fatal(err)
	}

	runTimeClient := pb.NewRuntimeServiceClient(conn)

	client := &util.ClientType{imageclient, runTimeClient}

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
		AbsPath(util.ApiPath).
		Resource("imagelists").
		Name(*imageListPtr).
		Do(context.Background()).Into(&result)

	if err != nil {
		log.Println("Unable to find imagelist", " Name: "+*imageListPtr, " AbsPath: ", util.ApiPath)
		log.Fatal(err)
	}

	// set target images to imagelist values
	targetImages = result.Spec.Images

	err = removeImages(clientset, client, socketPath, targetImages)

	if err != nil {
		log.Fatal(err)
	}
}
