package main

import (
	"context"

	util "github.com/Azure/eraser/pkg/utils"
)

func removeImages(c Client, targetImages []string) error {
	backgroundContext, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	images, err := c.listImages(backgroundContext)
	if err != nil {
		return err
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
		return err
	}

	// Images that are running
	// map of (digest | tag) -> digest
	runningImages := util.GetRunningImages(containers, idToTagListMap)

	// Images that aren't running
	// map of (digest | tag) -> digest
	nonRunningImages := util.GetNonRunningImages(runningImages, allImages, idToTagListMap)

	// Debug logs
	log.V(1).Info("Map of non-running images", "nonRunningImages", nonRunningImages)
	log.V(1).Info("Map of running images", "runningImages", runningImages)
	log.V(1).Info("Map of digest to image name(s)", "idToTaglistMap", idToTagListMap)

	// remove target images
	var prune bool
	deletedImages := make(map[string]struct{}, len(targetImages))
	for _, imgDigestOrTag := range targetImages {
		if imgDigestOrTag == "*" {
			prune = true
			continue
		}

		if digest, isNonRunning := nonRunningImages[imgDigestOrTag]; isNonRunning {
			if ex := util.IsExcluded(excluded, imgDigestOrTag, idToTagListMap); ex {
				log.Info("Image is excluded", "given", imgDigestOrTag, "digest", digest, "name", idToTagListMap[digest])
				continue
			}

			err = c.deleteImage(backgroundContext, digest)
			if err != nil {
				log.Error(err, "Error removing", "given", imgDigestOrTag, "digest", digest, "name", idToTagListMap[digest])
				continue
			}

			deletedImages[imgDigestOrTag] = struct{}{}
			log.Info("Removed", "given", imgDigestOrTag, "digest", digest, "name", idToTagListMap[digest])
			continue
		}

		digest, isRunning := runningImages[imgDigestOrTag]
		if isRunning {
			log.Info("Image is running", "given", imgDigestOrTag, "digest", digest, "name", idToTagListMap[digest])
			continue
		}

		log.Info("Image is not on node", "given", imgDigestOrTag)
	}

	if prune {
		for _, digest := range nonRunningImages {
			if _, deleted := deletedImages[digest]; deleted {
				continue
			}

			if util.IsExcluded(excluded, digest, idToTagListMap) {
				log.Info("Image is excluded", "digest", digest, "name", idToTagListMap[digest])
				continue
			}

			if err := c.deleteImage(backgroundContext, digest); err != nil {
				log.Error(err, "Error during prune", "digest", digest, "name", idToTagListMap[digest])
				continue
			}
			log.Info("Prune successful", "digest", digest, "name", idToTagListMap[digest])
			deletedImages[digest] = struct{}{}
		}
	}

	return nil
}
