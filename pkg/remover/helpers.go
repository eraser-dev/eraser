package main

import (
	"context"

	"github.com/eraser-dev/eraser/api/unversioned"
	"github.com/eraser-dev/eraser/pkg/cri"
	util "github.com/eraser-dev/eraser/pkg/utils"
)

// deleteImage gives one image its own budget. Deletion cost tracks unique layer
// bytes rather than reported size -- measured at 15s, 54s and 74s for three
// similarly sized Windows images -- so a whole-run budget cannot be sized from
// the image list up front.
func deleteImage(ctx context.Context, c cri.Remover, imageID string) error {
	deleteCtx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	return c.DeleteImage(deleteCtx, imageID)
}

func removeImages(ctx context.Context, c cri.Remover, targetImages []string) (int, error) {
	removed := 0

	// The listing calls get their own budget, separate from the deletions below.
	// Sharing one meant a slow node could spend the whole allowance before the
	// first image was deleted.
	listCtx, cancelList := context.WithTimeout(ctx, listTimeout)
	defer cancelList()

	images, err := c.ListImages(listCtx)
	if err != nil {
		return 0, err
	}

	allImages := make([]unversioned.Image, 0, len(images))
	// map with key: imageID, value: repoTag list (contains full name of image)
	idToImageMap := make(map[string]unversioned.Image)

	for _, img := range images {
		repoTags := []string{}
		repoTags = append(repoTags, img.RepoTags...)

		newImg := unversioned.Image{
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

	containers, err := c.ListContainers(listCtx)
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
		// Without this the loop keeps going once the caller is gone, logging one
		// instant failure per remaining image and still reporting partial success.
		if err := ctx.Err(); err != nil {
			log.Error(err, "stopping before the image list was finished", "removed", removed, "remaining", len(targetImages)-removed)
			return removed, err
		}

		if imgDigestOrTag == "*" {
			prune = true
			continue
		}

		if imageID, isNonRunning := nonRunningImages[imgDigestOrTag]; isNonRunning {
			if ex := util.IsExcluded(excluded, imgDigestOrTag, idToImageMap); ex {
				log.Info("image is excluded", "given", imgDigestOrTag, "imageID", imageID, "name", idToImageMap[imageID])
				continue
			}

			err = deleteImage(ctx, c, imageID)
			if err != nil {
				log.Error(err, "error removing image", "given", imgDigestOrTag, "imageID", imageID, "name", idToImageMap[imageID])
				continue
			}

			deletedImages[imgDigestOrTag] = struct{}{}
			log.Info("removed image", "given", imgDigestOrTag, "imageID", imageID, "name", idToImageMap[imageID])
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
			if err := ctx.Err(); err != nil {
				log.Error(err, "stopping before the prune was finished", "removed", removed)
				return removed, err
			}

			if _, deleted := deletedImages[imageID]; deleted {
				continue
			}

			if util.IsExcluded(excluded, imageID, idToImageMap) {
				log.Info("image is excluded", "imageID", imageID, "name", idToImageMap[imageID])
				continue
			}

			if err := deleteImage(ctx, c, imageID); err != nil {
				success = false
				log.Error(err, "error removing image", "imageID", imageID, "name", idToImageMap[imageID])
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
