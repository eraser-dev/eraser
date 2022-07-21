package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/Azure/eraser/pkg/logger"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	util "github.com/Azure/eraser/pkg/utils"
)

const (
	apiPath      = "apis/eraser.sh/v1alpha1"
	excludedPath = "/run/eraser.sh/excluded/excluded"
)

var (
	runtimePtr    = flag.String("runtime", "containerd", "container runtime")
	enableProfile = flag.Bool("enable-pprof", false, "enable pprof profiling")
	profilePort   = flag.Int("pprof-port", 6060, "port for pprof profiling. defaulted to 6060 if unspecified")

	// Timeout  of connecting to server (default: 5m).
	timeout  = 5 * time.Minute
	log      = logf.Log.WithName("collector")
	excluded map[string]struct{}
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
		fmt.Fprintln(os.Stderr, "Error setting up logger:", err)
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

	runTimeClient := pb.NewRuntimeServiceClient(conn)
	client := &client{imageclient, runTimeClient}

	excluded, err = util.ParseExcluded(excludedPath)
	if err != nil {
		log.Error(err, "failed to parse exclusion list")
		os.Exit(1)
	}
	if len(excluded) == 0 {
		log.Info("excluded configmap was empty or does not exist")
	}

	// finalImages of type []Image
	finalImages, err := getImages(client)
	if err != nil {
		log.Error(err, "failed to list all images")
		os.Exit(1)
	}

	test, err := json.Marshal(finalImages)
	if err != nil {
		log.Error(err, "failed to encode finalImages")
		os.Exit(1)
	}

	// fileMode 0777 = public read write
	if err := os.WriteFile("/run/eraser.sh/shared-data/all-images", test, os.FileMode(0777)); err != nil {
		log.Error(err, "failed to write to shared-data")
		os.Exit(1)
	}

	/*
		file, err := os.OpenFile("eraserPipe", os.O_RDWR, os.ModeNamedPipe)

		if _, err := file.Write(test); err != nil {
			log.Error(err, "filed to write to eraserPipe")
			os.Exit(1)
		}

		if err := file.Close(); err != nil {
			log.Error(err, "failed to close eraserPipe")
			os.Exit(1)
		} */

	/*	if err := createCollectorCR(context.Background(), finalImages); err != nil {
		log.Error(err, "Error creating ImageCollector CR")
		os.Exit(1)
	} */
}
