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
	generalErr = 1
)

func main() {
	flag.Parse()

	if *enableProfile {
		go func() {
			server := &http.Server{
				Addr:              fmt.Sprintf("localhost:%d", *profilePort),
				ReadHeaderTimeout: 3 * time.Second,
			}
			err := server.ListenAndServe()
			log.Error(err, "pprof server failed")
		}()
	}

	if err := logger.Configure(); err != nil {
		fmt.Fprintln(os.Stderr, "error setting up logger:", err)
		os.Exit(generalErr)
	}

	socketPath, found := util.RuntimeSocketPathMap[*runtimePtr]
	if !found {
		log.Error(fmt.Errorf("unsupported runtime"), "runtime", *runtimePtr)
		os.Exit(generalErr)
	}

	imageclient, conn, err := util.GetImageClient(context.Background(), socketPath)
	if err != nil {
		log.Error(err, "failed to get image client")
		os.Exit(generalErr)
	}

	runtimeClient := pb.NewRuntimeServiceClient(conn)
	client := client{imageclient, runtimeClient}

	var imagelist []string

	if *imageListPtr == "" {
		var f *os.File
		for {
			var err error

			f, err = os.OpenFile(util.ScanErasePath, os.O_RDONLY, 0)
			if err == nil {
				break
			}
			if !os.IsNotExist(err) {
				log.Error(err, "error opening scanErase pipe")
				os.Exit(generalErr)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		// json data is list of []eraserv1alpha1.Image
		data, err := io.ReadAll(f)
		if err != nil {
			log.Error(err, "error reading non-compliant images")
			os.Exit(generalErr)
		}

		nonCompliantImages := []eraserv1alpha1.Image{}
		if err = json.Unmarshal(data, &nonCompliantImages); err != nil {
			log.Error(err, "error in unmarshal non-compliant images")
			os.Exit(generalErr)
		}

		for _, img := range nonCompliantImages {
			imagelist = append(imagelist, img.Digest)
		}

		log.Info("successfully created imagelist from scanned non-compliant images")
	} else {
		imagelist, err = util.ParseImageList(*imageListPtr)
		if err != nil {
			log.Error(err, "failed to parse image list file")
			os.Exit(generalErr)
		}
		log.Info("successfully parsed image list file")
	}

	excluded, err = util.ParseExcluded()
	if os.IsNotExist(err) {
		log.Info("configmaps for exclusion do not exist")
	} else if err != nil {
		log.Error(err, "failed to parse exclusion list")
		os.Exit(generalErr)
	}
	if len(excluded) == 0 {
		log.Info("no images to exclude")
	}

	if err := removeImages(&client, imagelist); err != nil {
		log.Error(err, "failed to remove images")
		os.Exit(generalErr)
	}
}
