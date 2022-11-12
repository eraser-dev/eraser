package main

import (
	"context"
	"strings"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	util "github.com/Azure/eraser/pkg/utils"
)

func removeImages(c Client, targetImages []string) (int, error) {
	removed := 0

	backgroundContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	images, err := c.listImages(backgroundContext)
	if err != nil {
		return 0, err
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
				log.Info("repoDigest not formatted as image@digest", "repodigest", repoDigest)
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
		return 0, err
	}

	// Images that are running
	// map of (digest | name) -> imageID
	runningImages := util.GetRunningImages(containers, idToImageMap)

	// Images that aren't running
	// map of (digest | name) -> imageID
	nonRunningImages := util.GetNonRunningImages(runningImages, allImages, idToImageMap)

	// Debug logs
	log.V(1).Info("Map of non-running images", "nonRunningImages", nonRunningImages)
	log.V(1).Info("Map of running images", "runningImages", runningImages)
	log.V(1).Info("Map of digest to image name(s)", "idToImageMap", idToImageMap)

	// remove target images
	var prune bool
	deletedImages := make(map[string]struct{}, len(targetImages))
	for _, imgDigestOrTag := range targetImages {
		if imgDigestOrTag == "*" {
			prune = true
			continue
		}

		if imageID, isNonRunning := nonRunningImages[imgDigestOrTag]; isNonRunning {
			if ex := util.IsExcluded(excluded, imgDigestOrTag, idToImageMap); ex {
				log.Info("image is excluded", "given", imgDigestOrTag, "imageID", imageID)
				continue
			}

			err = c.deleteImage(backgroundContext, imageID)
			if err != nil {
				log.Error(err, "error removing image", "given", imgDigestOrTag, "imageID", imageID)
				continue
			}

			deletedImages[imgDigestOrTag] = struct{}{}
			log.Info("removed image", "given", imgDigestOrTag, "imageID", imageID)
			removed++
			continue
		}

		imageID, isRunning := runningImages[imgDigestOrTag]
		if isRunning {
			log.Info("image is running", "given", imgDigestOrTag, "imageID", imageID, "name", idToImageMap[imageID])
			continue
		}

		log.Info("image is not on node", "given", imgDigestOrTag)
	}

	if prune {
		success := true
		for _, imageID := range nonRunningImages {
			if _, deleted := deletedImages[imageID]; deleted {
				continue
			}

			if util.IsExcluded(excluded, imageID, idToImageMap) {
				log.Info("image is excluded", "imageID", imageID)
				continue
			}

			if err := c.deleteImage(backgroundContext, imageID); err != nil {
				success = false
				log.Error(err, "error removing image", "imageID", imageID)
				continue
			}

			log.Info("removed image", "digest", imageID)
			deletedImages[imageID] = struct{}{}
			removed++
		}
		if success {
			log.Info("prune successful")
		} else {
			log.Info("error during prune")
		}
	}

	return removed, nil
}
