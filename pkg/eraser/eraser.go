package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"regexp"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Azure/eraser/pkg/logger"

	util "github.com/Azure/eraser/pkg/utils"
)

var (
	// Timeout  of connecting to server (default: 5m).
	timeout  = 5 * time.Minute
	log      = logf.Log.WithName("eraser")
	excluded map[string]struct{}
)

const (
	excludedPath = "/run/eraser.sh/excluded/excluded"
)

type client struct {
	images  pb.ImageServiceClient
	runtime pb.RuntimeServiceClient
}

type Client interface {
	listImages(context.Context) ([]*pb.Image, error)
	listContainers(context.Context) ([]*pb.Container, error)
	deleteImage(context.Context, string) error
}

func (c *client) listContainers(ctx context.Context) (list []*pb.Container, err error) {
	return util.ListContainers(ctx, c.runtime)
}

func (c *client) listImages(ctx context.Context) (list []*pb.Image, err error) {
	return util.ListImages(ctx, c.images)
}

func (c *client) deleteImage(ctx context.Context, image string) (err error) {
	if image == "" {
		return err
	}

	request := &pb.RemoveImageRequest{Image: &pb.ImageSpec{Image: image}}

	_, err = c.images.RemoveImage(ctx, request)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return err
	}

	return nil
}

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

func main() {
	runtimePtr := flag.String("runtime", "containerd", "container runtime")
	imageListPtr := flag.String("imagelist", "", "name of ImageList")
	enableProfile := flag.Bool("enable-pprof", false, "enable pprof profiling")
	profilePort := flag.Int("pprof-port", 6060, "port for pprof profiling. defaulted to 6060 if unspecified")

	if *enableProfile {
		go func() {
			err := http.ListenAndServe(fmt.Sprintf("localhost:%d", *profilePort), nil)
			log.Error(err, "pprof server failed")
		}()
	}

	flag.Parse()

	if err := logger.Configure(); err != nil {
		fmt.Fprintln(os.Stderr, "Error setting up logger:", err)
		os.Exit(1)
	}

	var socketPath string

	switch runtime := *runtimePtr; runtime {
	case "docker":
		socketPath = "unix:///var/run/dockershim.sock"
	case "containerd":
		socketPath = "unix:///run/containerd/containerd.sock"
	case "cri-o":
		socketPath = "unix:///var/run/crio/crio.sock"
	default:
		log.Error(fmt.Errorf("unsupported runtime"), "runtime", runtime)
		os.Exit(1)
	}

	imageclient, conn, err := util.GetImageClient(context.Background(), socketPath)
	if err != nil {
		log.Error(err, "failed to get image client")
		os.Exit(1)
	}

	runTimeClient := pb.NewRuntimeServiceClient(conn)

	client := &client{imageclient, runTimeClient}

	data, err := os.ReadFile(*imageListPtr)
	if err != nil {
		log.Error(err, "failed to read image list file")
		os.Exit(1)
	}

	var ls []string
	if err := json.Unmarshal(data, &ls); err != nil {
		log.Error(err, "failed to unmarshal image list")
		os.Exit(1)
	}

	// read excluded values from excluded configmap
	data, err = os.ReadFile(excludedPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("excluded configmap does not exist", "error: ", err)
		} else {
			log.Error(err, "failed to read excluded values")
			os.Exit(1)
		}
	} else {
		var result map[string][]string
		if err := json.Unmarshal(data, &result); err != nil {
			log.Error(err, "failed to unmarshal excluded configmap")
			os.Exit(1)
		}

		excluded = make(map[string]struct{}, len(result))
		for _, img := range result["excluded"] {
			excluded[img] = struct{}{}
		}
	}

	if err := removeImages(client, ls); err != nil {
		log.Error(err, "failed to remove images")
		os.Exit(1)
	}
}
