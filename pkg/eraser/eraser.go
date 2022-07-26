package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Azure/eraser/pkg/logger"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	util "github.com/Azure/eraser/pkg/utils"
)

var (
	runtimePtr    = flag.String("runtime", "containerd", "container runtime")
	imageListPtr  = flag.String("imagelist", "", "name of ImageList")
	enableProfile = flag.Bool("enable-pprof", false, "enable pprof profiling")
	profilePort   = flag.Int("pprof-port", 6060, "port for pprof profiling. defaulted to 6060 if unspecified")

	// Timeout  of connecting to server (default: 5m).
	timeout  = 5 * time.Minute
	log      = logf.Log.WithName("eraser")
	excluded map[string]struct{}
)

const (
	excludedPath = "/run/eraser.sh/excluded/excluded"
)

func main() {
	flag.Parse()
	if *enableProfile {
		go func() {
			err := http.ListenAndServe(fmt.Sprintf("localhost:%d", *profilePort), nil)
			log.Error(err, "pprof server failed")
		}()
	}

	if err := logger.Configure(); err != nil {
		fmt.Fprintln(os.Stderr, "error setting up logger:", err)
		os.Exit(1)
	}

	socketPath, found := util.RuntimeSocketPathMap[*runtimePtr]
	if !found {
		log.Error(fmt.Errorf("unsupported runtime"), "runtime", *runtimePtr)
		os.Exit(1)
	}

	imageclient, conn, err := util.GetImageClient(context.Background(), socketPath)
	if err != nil {
		log.Error(err, "failed to get image client")
		os.Exit(1)
	}

	runtimeClient := pb.NewRuntimeServiceClient(conn)
	client := client{imageclient, runtimeClient}

	var imagelist []string

	if *imageListPtr == "" {
		fileR, err := os.OpenFile("/run/eraser.sh/shared-data/scanErase", os.O_RDONLY, os.ModeNamedPipe)
		if err != nil {
			log.Error(err, "error opening scanErase RD")
			os.Exit(1)
		}

		// json data is list of []eraserv1alpha1.Image
		data, err := io.ReadAll(fileR)
		if err != nil {
			log.Error(err, "error reading vulnerableImages")
			os.Exit(1)
		}

		vulnerableImages := &[]eraserv1alpha1.Image{}
		if err = json.Unmarshal(data, vulnerableImages); err != nil {
			log.Error(err, "error in unmarshal vulnerableImages")
			os.Exit(1)
		}

		for _, img := range *vulnerableImages {
			imagelist = append(imagelist, img.Digest)
		}

		log.Info("successfully created imagelist from scanned vulnerableImages")
	} else {
		imagelist, err = util.ParseImageList(*imageListPtr)
		if err != nil {
			log.Error(err, "failed to parse image list file")
			os.Exit(1)
		}
		log.Info("sucessfully parsed image list file")
	}

	excluded, err = util.ParseExcluded(excludedPath)
	if err != nil {
		log.Error(err, "failed to parse exclusion list")
		os.Exit(1)
	}
	if len(excluded) == 0 {
		log.Info("excluded configmap was empty or does not exist")
	}

	if err := removeImages(&client, imagelist); err != nil {
		log.Error(err, "failed to remove images")
		os.Exit(1)
	}
}
