package main

import (
	"context"
	"regexp"
	"strings"

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
			if ex := isExcluded(imgDigestOrTag, idToTagListMap); ex {
				log.Info("Image is excluded", "image", imgDigestOrTag)
				continue
			}

			err = c.deleteImage(backgroundContext, digest)
			if err != nil {
				log.Error(err, "Error removing", "image", digest)
				continue
			}

			deletedImages[imgDigestOrTag] = struct{}{}
			log.Info("Removed", "given", imgDigestOrTag, "digest", digest, "digest", digest)
			continue
		}

		_, isRunning := runningImages[imgDigestOrTag]
		if isRunning {
			log.Info("Image is running", "image", imgDigestOrTag)
			continue
		}

		log.Info("Image is not on node", "image", imgDigestOrTag)
	}

	if prune {
		for img := range nonRunningImages {
			if _, deleted := deletedImages[img]; deleted {
				continue
			}

			if _, running := runningImages[img]; running {
				continue
			}

			if ex := isExcluded(img, idToTagListMap); ex {
				log.Info("Image is excluded", "image", img)
				continue
			}

			if err := c.deleteImage(backgroundContext, img); err != nil {
				log.Error(err, "Error during prune", "image", img)
				continue
			}
			log.Info("Prune successful", "image", img)
		}
	}

	return nil
}

func isExcluded(img string, idToTagListMap map[string][]string) bool {
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

	regexRepo := regexp.MustCompile(`[a-z0-9]+([._-][a-z0-9]+)*/\*\z`)
	regexTag := regexp.MustCompile(`[a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*:\*\z`)

	// look for excluded repository values and names without tag
	for key := range excluded {
		// if excluded key ends in /*, check image with pattern match
		if match := regexRepo.MatchString(key); match {
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
		if match := regexTag.MatchString(key); match {
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
