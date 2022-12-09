package template

import (
	"context"
	"flag"
	"io"
	"os"
	"os/signal"
	"syscall"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/go-logr/logr"
	"golang.org/x/sys/unix"

	_ "net/http/pprof"

	"github.com/Azure/eraser/pkg/metrics"
	util "github.com/Azure/eraser/pkg/utils"
	"go.opentelemetry.io/otel/metric/global"
)

type ScannerTemplate interface {
	Initialize() []eraserv1alpha1.Image
	SendToEraser(vulnerableImages, failedImages []eraserv1alpha1.Image)
	Cleanup()
}

type config struct {
	ctx                    context.Context
	log                    logr.Logger
	deleteScanFailedImages bool
	reportMetrics          bool
}

type configFunc func(*config)

func (cfg *config) Initialize(funcs ...configFunc) []eraserv1alpha1.Image {
	for _, f := range funcs {
		f(cfg)
	}

	flag.Parse()
	var err error

	if err := unix.Mkfifo(util.EraseCompleteScanPath, util.PipeMode); err != nil {
		cfg.log.Error(err, "failed to create pipe", "pipeName", util.EraseCompleteScanPath)
		os.Exit(1)
	}

	err = os.Chmod(util.EraseCompleteScanPath, 0o666)
	if err != nil {
		cfg.log.Error(err, "unable to enable pipe for writing", "pipeName", util.EraseCompleteScanPath)
		os.Exit(1)
	}

	allImages, err := util.ReadCollectScanPipe(cfg.ctx)
	if err != nil {
		cfg.log.Error(err, "unable to read images from collect scan pipe")
		os.Exit(1)
	}

	return allImages
}

func (cfg *config) SendToEraser(vulnerableImages, failedImages []eraserv1alpha1.Image) {
	if cfg.deleteScanFailedImages {
		vulnerableImages = append(vulnerableImages, failedImages...)
	}

	if err := util.WriteScanErasePipe(vulnerableImages); err != nil {
		cfg.log.Error(err, "unable to write non-compliant images to scan erase pipe")
		os.Exit(1)
	}

	if cfg.reportMetrics {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		exporter, reader, provider := metrics.ConfigureMetrics(ctx, cfg.log, os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
		global.SetMeterProvider(provider)

		defer metrics.ExportMetrics(cfg.log, exporter, reader, provider)

		if err := metrics.RecordMetricsScanner(ctx, global.MeterProvider(), len(vulnerableImages)); err != nil {
			cfg.log.Error(err, "error recording metrics")
		}
	}
}

func (cfg *config) Cleanup() {
	file, err := os.OpenFile(util.EraseCompleteScanPath, os.O_RDONLY, 0)
	if err != nil {
		cfg.log.Error(err, "failed to open pipe", "pipeName", util.EraseCompleteScanPath)
		os.Exit(1)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		cfg.log.Error(err, "failed to read pipe", "pipeName", util.EraseCompleteScanPath)
		os.Exit(1)
	}

	file.Close()

	if string(data) != util.EraseCompleteMessage {
		cfg.log.Info("garbage in pipe", "pipeName", util.EraseCompleteScanPath, "in_pipe", string(data))
		os.Exit(1)
	}

	cfg.log.Info("scanning complete, exiting")
}

func WithContext(ctx context.Context) configFunc {
	return func(cfg *config) {
		cfg.ctx = ctx
	}
}

func WithdeleteScanFailedImages(deleteScanFailedImages bool) configFunc {
	return func(cfg *config) {
		cfg.deleteScanFailedImages = deleteScanFailedImages
	}
}

func WithLogger(log logr.Logger) configFunc {
	return func(cfg *config) {
		cfg.log = log
	}
}

func WithMetrics(reportMetrics bool) configFunc {
	return func(cfg *config) {
		cfg.reportMetrics = reportMetrics
	}
}
