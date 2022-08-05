package main

import (
	"context"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	util "github.com/Azure/eraser/pkg/utils"
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
