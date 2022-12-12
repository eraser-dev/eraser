package template

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/go-logr/logr"
	"golang.org/x/sys/unix"

	"github.com/Azure/eraser/pkg/metrics"
	util "github.com/Azure/eraser/pkg/utils"
	"go.opentelemetry.io/otel/metric/global"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// interface for custom scanners to communicate with Eraser
type ImageProvider interface {
	// receive list of all non-running, non-excluded images from collector container to process
	ReceiveImages() []eraserv1alpha1.Image

	// sends non-compliant images found to eraser container for removal
	SendImages(nonCompliantImages, failedImages []eraserv1alpha1.Image)

	// completes scanner communication process - required after custom scanning finishes
	Finish()
}

type config struct {
	ctx                    context.Context
	log                    logr.Logger
	deleteScanFailedImages bool
	reportMetrics          bool
}

type ConfigFunc func(*config)

func NewImageProvider(funcs ...ConfigFunc) ImageProvider {
	// default config
	cfg := &config{
		ctx:                    context.Background(),
		log:                    logf.Log.WithName("scanner"),
		deleteScanFailedImages: true,
		reportMetrics:          false,
	}

	// apply user config
	for _, f := range funcs {
		f(cfg)
	}

	return cfg
}

func (cfg *config) ReceiveImages() []eraserv1alpha1.Image {
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

func (cfg *config) SendImages(nonCompliantImages, failedImages []eraserv1alpha1.Image) {
	if cfg.deleteScanFailedImages {
		nonCompliantImages = append(nonCompliantImages, failedImages...)
	}

	if err := util.WriteScanErasePipe(nonCompliantImages); err != nil {
		cfg.log.Error(err, "unable to write non-compliant images to scan erase pipe")
		os.Exit(1)
	}

	if cfg.reportMetrics {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		exporter, reader, provider := metrics.ConfigureMetrics(ctx, cfg.log, os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
		global.SetMeterProvider(provider)

		defer metrics.ExportMetrics(cfg.log, exporter, reader, provider)

		if err := metrics.RecordMetricsScanner(ctx, global.MeterProvider(), len(nonCompliantImages)); err != nil {
			cfg.log.Error(err, "error recording metrics")
		}
	}
}

func (cfg *config) Finish() {
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

// provide custom context
func WithContext(ctx context.Context) ConfigFunc {
	return func(cfg *config) {
		cfg.ctx = ctx
	}
}

// sets deleteScanFailedImages flag
func WithDeleteScanFailedImages(deleteScanFailedImages bool) ConfigFunc {
	return func(cfg *config) {
		cfg.deleteScanFailedImages = deleteScanFailedImages
	}
}

// provide custom logger
func WithLogger(log logr.Logger) ConfigFunc {
	return func(cfg *config) {
		cfg.log = log
	}
}

// sets boolean for recording metrics
func WithMetrics(reportMetrics bool) ConfigFunc {
	return func(cfg *config) {
		cfg.reportMetrics = reportMetrics
	}
}
