package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"golang.org/x/sys/unix"

	_ "net/http/pprof"

	"github.com/Azure/eraser/pkg/logger"
	"github.com/Azure/eraser/pkg/metrics"
	util "github.com/Azure/eraser/pkg/utils"
	"go.opentelemetry.io/otel/metric/global"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	generalErr = 1
	// name of custom scanner.
	providerName = "CUSTOM_SCANNER"
)

var (
	log                    = logf.Log.WithName("scanner").WithValues("provider", providerName)
	deleteScanFailedImages = flag.Bool("delete-scan-failed-images", true, "whether or not to delete images for which scanning has failed")
)

func main() {
	// setup
	flag.Parse()
	ctx := context.Background()

	var err error

	if err = logger.Configure(); err != nil {
		fmt.Fprintln(os.Stderr, "error setting up logger:", err)
		os.Exit(generalErr)
	}

	if err := unix.Mkfifo(util.EraseCompleteScanPath, util.PipeMode); err != nil {
		log.Error(err, "failed to create pipe", "pipeName", util.EraseCompleteScanPath)
		os.Exit(1)
	}

	err = os.Chmod(util.EraseCompleteScanPath, 0o666)
	if err != nil {
		log.Error(err, "unable to enable pipe for writing", "pipeName", util.EraseCompleteScanPath)
		os.Exit(1)
	}

	// list of allImages read by collector container
	allImages, err := util.ReadCollectScanPipe(ctx)
	if err != nil {
		log.Error(err, "unable to read images from collect scan pipe")
		os.Exit(generalErr)
	}

	/* TODO: implement customized scanner to scan allImages and partition into vulnerableImages and failedImages */
	vulnerableImages := make([]eraserv1alpha1.Image, 0, len(allImages))
	failedImages := make([]eraserv1alpha1.Image, 0, len(allImages))

	// if deleteScanFailedImages is true, we want to pass failed images as vulnerable to be deleted
	if *deleteScanFailedImages {
		vulnerableImages = append(vulnerableImages, failedImages...)
	}

	// write vulnerable images to scanErase pipe for eraser container to read
	if err := util.WriteScanErasePipe(vulnerableImages); err != nil {
		log.Error(err, "unable to write non-compliant images to scan erase pipe")
		os.Exit(generalErr)
	}

	file, err := os.OpenFile(util.EraseCompleteScanPath, os.O_RDONLY, 0)
	if err != nil {
		log.Error(err, "failed to open pipe", "pipeName", util.EraseCompleteScanPath)
		os.Exit(1)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		log.Error(err, "failed to read pipe", "pipeName", util.EraseCompleteScanPath)
		os.Exit(1)
	}

	file.Close()

	if string(data) != util.EraseCompleteMessage {
		log.Info("garbage in pipe", "pipeName", util.EraseCompleteScanPath, "in_pipe", string(data))
		os.Exit(1)
	}

	// record scanner metrics
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		exporter, reader, provider := metrics.ConfigureMetrics(ctx, log, os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
		global.SetMeterProvider(provider)

		defer metrics.ExportMetrics(log, exporter, reader, provider)

		if err := metrics.RecordMetricsScanner(ctx, global.MeterProvider(), len(vulnerableImages)); err != nil {
			log.Error(err, "error recording metrics")
		}
	}

	log.Info("scanning complete, exiting")
}
