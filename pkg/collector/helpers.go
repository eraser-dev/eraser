package main

import (
	"context"
	"strings"

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
		repoTags := img.RepoTags
		if len(repoTags) == 0 {
			repoTags = []string{}
		}

		newImg := eraserv1alpha1.Image{
			ImageID: img.Id,
			Names:   repoTags,
		}

		digests := make(map[string]struct{})
		for _, repoDigest := range img.RepoDigests {
			s := strings.Split(repoDigest, "@")
			if len(s) < 2 {
				log.Info("repoDigest not formatted as image@digest", "repodigest", "repoDigest")
				continue
			}
			digest := s[1]
			digests[digest] = struct{}{}
		}

		for digest := range digests {
			newImg.Digests = append(newImg.Digests, digest)
		}

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
		if _, alreadyChecked := checked[imageID]; !alreadyChecked {
			checked[imageID] = struct{}{}

			// set the current image's digest?
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
	}

	return finalImages, nil
}
