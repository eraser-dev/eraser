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
	"golang.org/x/sys/unix"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	util "github.com/Azure/eraser/pkg/utils"
)

var (
	runtimePtr    = flag.String("runtime", "containerd", "container runtime")
	enableProfile = flag.Bool("enable-pprof", false, "enable pprof profiling")
	profilePort   = flag.Int("pprof-port", 6060, "port for pprof profiling. defaulted to 6060 if unspecified")
	scanDisabled  = flag.Bool("scan-disabled", false, "boolean for if scanner container is disabled")

	// Timeout  of connecting to server (default: 5m).
	timeout  = 5 * time.Minute
	log      = logf.Log.WithName("collector")
	excluded map[string]struct{}
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

	excluded, err = util.ParseExcluded()
	if os.IsNotExist(err) {
		log.Info("configmaps for exclusion do not exist")
	} else if err != nil {
		log.Error(err, "failed to parse exclusion list")
		os.Exit(1)
	}
	if len(excluded) == 0 {
		log.Info("no images to exclude")
	}

	// finalImages of type []Image
	finalImages, err := getImages(client)
	if err != nil {
		log.Error(err, "failed to list all images")
		os.Exit(1)
	}

	data, err := json.Marshal(finalImages)
	if err != nil {
		log.Error(err, "failed to encode finalImages")
		os.Exit(1)
	}

	if *scanDisabled {
		if err := unix.Mkfifo(util.ScanErasePath, util.PipeMode); err != nil {
			log.Error(err, "failed to create scanErase pipe")
			os.Exit(1)
		}

		file, err := os.OpenFile(util.ScanErasePath, os.O_WRONLY, 0)
		if err != nil {
			log.Error(err, "failed to open scanErase pipe")
			os.Exit(1)
		}

		if _, err := file.Write(data); err != nil {
			log.Error(err, "failed to write to scanErase pipe")
			os.Exit(1)
		}

		file.Close()
	} else {
		if err := unix.Mkfifo(util.CollectScanPath, util.PipeMode); err != nil {
			log.Error(err, "failed to create collectScan pipe")
			os.Exit(1)
		}

		file, err := os.OpenFile(util.CollectScanPath, os.O_WRONLY, 0)
		if err != nil {
			log.Error(err, "failed to open collectScan pipe")
			os.Exit(1)
		}

		if _, err := file.Write(data); err != nil {
			log.Error(err, "failed to write to collectScan pipe")
			os.Exit(1)
		}

		file.Close()
	}
}
