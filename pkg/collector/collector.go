package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/eraser-dev/eraser/pkg/cri"
	"github.com/eraser-dev/eraser/pkg/logger"
	"golang.org/x/sys/unix"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	util "github.com/eraser-dev/eraser/pkg/utils"
)

var (
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

	client, err := cri.NewCollectorClient(util.CRIPath)
	if err != nil {
		log.Error(err, "failed to get image client")
		os.Exit(1)
	}

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
	log.Info("images collected", "finalImages:", finalImages)

	data, err := json.Marshal(finalImages)
	if err != nil {
		log.Error(err, "failed to encode finalImages")
		os.Exit(1)
	}

	path := util.CollectScanPath

	if *scanDisabled {
		path = util.ScanErasePath
	}

	if err := unix.Mkfifo(path, util.PipeMode); err != nil {
		log.Error(err, "failed to create pipe", "pipeFile", path)
		os.Exit(1)
	}

	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		log.Error(err, "failed to open pipe", "pipeFile", path)
		os.Exit(1)
	}

	if _, err := file.Write(data); err != nil {
		log.Error(err, "failed to write to pipe", "pipeFile", path)
		os.Exit(1)
	}

	file.Close()
	if err := unix.Mkfifo(util.EraseCompleteCollectPath, util.PipeMode); err != nil {
		log.Error(err, "failed to create pipe", "pipeFile", util.EraseCompleteCollectPath)
		os.Exit(1)
	}

	file, err = os.OpenFile(util.EraseCompleteCollectPath, os.O_RDONLY, 0)
	if err != nil {
		log.Error(err, "failed to open pipe", "pipeFile", util.EraseCompleteCollectPath)
		os.Exit(1)
	}

	data, err = io.ReadAll(file)
	if err != nil {
		log.Error(err, "failed to read pipe", "pipeFile", util.EraseCompleteCollectPath)
		os.Exit(1)
	}

	file.Close()

	if string(data) != util.EraseCompleteMessage {
		log.Info("garbage in pipe", "pipeFile", util.EraseCompleteCollectPath, "in_pipe", string(data))
		os.Exit(1)
	}
}
