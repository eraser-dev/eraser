package main

import (
	"context"
	"encoding/json"
	"os"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	util "github.com/Azure/eraser/pkg/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func getImages(c Client) ([]eraserv1alpha1.Image, error) {
	backgroundContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	images, err := c.listImages(backgroundContext)
	if err != nil {
		return nil, err
	}

	allImages := make([]string, 0, len(images))

	// map with key: sha id, value: repoTag list (contains full name of image)
	idToTagListMap := make(map[string][]string)

	for _, img := range images {
		allImages = append(allImages, img.Id)
		idToTagListMap[img.Id] = img.RepoTags
	}

	containers, err := c.listContainers(backgroundContext)
	if err != nil {
		return nil, err
	}

	// Images that are running
	// map of (digest | tag) -> digest
	runningImages := util.GetRunningImages(containers, idToTagListMap)

	// Images that aren't running
	// map of (digest | tag) -> digest
	nonRunningImages := util.GetNonRunningImages(runningImages, allImages, idToTagListMap)

	finalImages := make([]eraserv1alpha1.Image, 0, len(images))

	// empty map to keep track of repeated digest values due to both name and digest being present as keys in nonRunningImages
	checked := make(map[string]struct{})

	for _, digest := range nonRunningImages {
		if _, exists := checked[digest]; !exists {
			checked[digest] = struct{}{}

			currImage := eraserv1alpha1.Image{
				Digest: digest,
			}

			if len(idToTagListMap[digest]) > 0 {
				currImage.Name = idToTagListMap[digest][0]
			}

			if !util.IsExcluded(excluded, currImage.Digest, idToTagListMap) {
				finalImages = append(finalImages, currImage)
			}
		}
	}

	return finalImages, nil
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
