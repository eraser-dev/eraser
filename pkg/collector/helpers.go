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

	allImages := make([]eraserv1alpha1.Image, 0, len(images))
	// map with key: imageID, value: repoTag list (contains full name of image)
	idToImageMap := make(map[string]eraserv1alpha1.Image)

	for _, img := range images {
		repoTags := []string{}
		repoTags = append(repoTags, img.RepoTags...)

		newImg := eraserv1alpha1.Image{
			ImageID: img.Id,
			Names:   repoTags,
		}

		digests, errs := util.ProcessRepoDigests(img.RepoDigests)
		for _, err := range errs {
			log.Error(err, "error processing digest")
		}

		newImg.Digests = append(newImg.Digests, digests...)

		allImages = append(allImages, newImg)
		idToImageMap[img.Id] = newImg
	}

	containers, err := c.listContainers(backgroundContext)
	if err != nil {
		return nil, err
	}

	// Images that are running
	// map of (digest | name) -> imageID
	runningImages := util.GetRunningImages(containers, idToImageMap)

	// Images that aren't running
	// map of (digest | name) -> imageID
	nonRunningImages := util.GetNonRunningImages(runningImages, allImages, idToImageMap)

	finalImages := make([]eraserv1alpha1.Image, 0, len(images))

	// empty map to keep track of repeated digest values due to both name and digest being present as keys in nonRunningImages
	checked := make(map[string]struct{})

	for _, imageID := range nonRunningImages {
		if _, alreadyChecked := checked[imageID]; alreadyChecked {
			continue
		}

		checked[imageID] = struct{}{}
		img := idToImageMap[imageID]

		currImage := eraserv1alpha1.Image{
			ImageID: imageID,
			Names:   img.Names,
			Digests: img.Digests,
		}

		if !util.IsExcluded(excluded, currImage.ImageID, idToImageMap) {
			finalImages = append(finalImages, currImage)
		}
	}

	return finalImages, nil
}
