package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eraser-dev/eraser/pkg/cri"
	"github.com/eraser-dev/eraser/pkg/logger"
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

	path := util.CollectScanPath

	if *scanDisabled {
		path = util.ScanErasePath
	}

	// Published before the payload, not after: the peer can finish and signal
	// back the moment it has read the list, so an endpoint created afterwards
	// can be missed entirely. The scanner already publishes in this order.
	completion, err := util.CreateCompletionPipe(util.EraseCompleteCollectPath)
	if err != nil {
		log.Error(err, "failed to create pipe", "pipeFile", util.EraseCompleteCollectPath)
		os.Exit(1)
	}

	// Both calls below observe ctx, so the handler can stay registered for the
	// rest of the process. The stop func is discarded rather than deferred
	// because every exit path here is os.Exit, which would skip it anyway.
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	if err := util.WriteImagesPipe(ctx, path, finalImages); err != nil {
		log.Error(err, "failed to send images", "pipeFile", path)
		os.Exit(1)
	}

	data, err := completion.Await(ctx)
	if err != nil {
		log.Error(err, "failed to read pipe", "pipeFile", util.EraseCompleteCollectPath)
		os.Exit(1)
	}

	if err := completion.Close(); err != nil {
		log.Error(err, "failed to close pipe", "pipeFile", util.EraseCompleteCollectPath)
		os.Exit(1)
	}

	if string(data) != util.EraseCompleteMessage {
		log.Info("garbage in pipe", "pipeFile", util.EraseCompleteCollectPath, "in_pipe", string(data))
		os.Exit(1)
	}
}
