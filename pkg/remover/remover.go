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

	"go.opentelemetry.io/otel"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/eraser-dev/eraser/pkg/cri"
	"github.com/eraser-dev/eraser/pkg/logger"
	"github.com/eraser-dev/eraser/pkg/metrics"

	util "github.com/eraser-dev/eraser/pkg/utils"
)

var (
	imageListPtr  = flag.String("imagelist", "", "name of ImageList")
	enableProfile = flag.Bool("enable-pprof", false, "enable pprof profiling")
	profilePort   = flag.Int("pprof-port", 6060, "port for pprof profiling. defaulted to 6060 if unspecified")

	// Timeout  of connecting to server (default: 5m).
	timeout  = 5 * time.Minute
	log      = logf.Log.WithName("remover")
	excluded map[string]struct{}
)

const (
	generalErr = 1
)

func main() {
	flag.Parse()

	// A terminating pod should not leave the worker blocked on a peer that is
	// never going to arrive. The stop func is discarded rather than deferred
	// because every exit path here is os.Exit, which would skip it anyway.
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

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

	log.Info("remover starting", "imageListPtr", *imageListPtr, "criPath", util.CRIPath)

	client, err := cri.NewRemoverClient(util.CRIPath)
	if err != nil {
		log.Error(err, "failed to get image client")
		os.Exit(generalErr)
	}

	log.Info("CRI client created successfully")

	var imagelist []string

	if *imageListPtr == "" {
		log.Info("imageListPtr is empty, waiting for scan erase path")
	} else {
		log.Info("imageListPtr provided, parsing image list file", "path", *imageListPtr)
	}

	if *imageListPtr == "" {
		nonCompliantImages, err := util.ReadImagesPipe(ctx, util.ScanErasePath)
		if err != nil {
			log.Error(err, "error reading non-compliant images")
			os.Exit(generalErr)
		}

		for _, img := range nonCompliantImages {
			imagelist = append(imagelist, img.ImageID)
		}

		log.Info("successfully created imagelist from scanned non-compliant images")
	} else {
		log.Info("attempting to parse image list file", "path", *imageListPtr)
		imagelist, err = util.ParseImageList(*imageListPtr)
		if err != nil {
			log.Error(err, "failed to parse image list file", "path", *imageListPtr)
			os.Exit(generalErr)
		}
		log.Info("successfully parsed image list file", "count", len(imagelist), "images", imagelist)
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

	removed, err := removeImages(ctx, client, imagelist)
	if err != nil {
		log.Error(err, "failed to remove images")
		os.Exit(generalErr)
	}

	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		// record metrics
		exporter, reader, provider := metrics.ConfigureMetrics(ctx, log, os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
		otel.SetMeterProvider(provider)

		if err := metrics.RecordMetricsRemover(ctx, otel.GetMeterProvider(), int64(removed)); err != nil {
			log.Error(err, "error recording metrics")
		}
		metrics.ExportMetrics(log, exporter, reader)
	}

	if *imageListPtr == "" {
		if err := util.WriteCompletionPipe(ctx, util.EraseCompleteCollectPath); err != nil {
			log.Error(err, "unable to signal completion", "pipeFile", util.EraseCompleteCollectPath)
			os.Exit(generalErr)
		}

		err := util.WriteCompletionPipe(ctx, util.EraseCompleteScanPath)
		// if the scanner is disabled
		if os.IsNotExist(err) {
			return
		}
		if err != nil {
			log.Error(err, "unable to signal completion", "pipeFile", util.EraseCompleteScanPath)
			os.Exit(generalErr)
		}
	}
}
