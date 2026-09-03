package template

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/eraser-dev/eraser/api/unversioned"
	"github.com/go-logr/logr"

	"github.com/eraser-dev/eraser/pkg/metrics"
	util "github.com/eraser-dev/eraser/pkg/utils"
	"go.opentelemetry.io/otel"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// interface for custom scanners to communicate with Eraser.
type ImageProvider interface {
	// receive list of all non-running, non-excluded images from collector container to process.
	ReceiveImages() ([]unversioned.Image, error)

	// sends non-compliant images found to remover container for removal.
	SendImages(nonCompliantImages, failedImages []unversioned.Image) error

	// completes scanner communication process - required after custom scanning finishes.
	Finish() error
}

type config struct {
	ctx                    context.Context
	log                    logr.Logger
	deleteScanFailedImages bool
	deleteEOLImages        bool
	reportMetrics          bool

	// held from ReceiveImages until Finish: the endpoint must stay published so
	// the remover can tell a scanner is present
	completion *util.CompletionPipe
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

func (cfg *config) ReceiveImages() ([]unversioned.Image, error) {
	var err error

	// published up front so the remover can tell a scanner is present
	cfg.completion, err = util.CreateCompletionPipe(util.EraseCompleteScanPath)
	if err != nil {
		cfg.log.Error(err, "failed to create pipe", "pipeName", util.EraseCompleteScanPath)
		return nil, err
	}

	err = os.Chmod(util.EraseCompleteScanPath, 0o600)
	if err != nil {
		cfg.log.Error(err, "unable to enable pipe for writing", "pipeName", util.EraseCompleteScanPath)
		return nil, err
	}

	allImages, err := util.ReadCollectScanPipe(cfg.ctx)
	if err != nil {
		cfg.log.Error(err, "unable to read images from collect scan pipe")
		return nil, err
	}

	return allImages, nil
}

func (cfg *config) SendImages(nonCompliantImages, failedImages []unversioned.Image) error {
	if cfg.deleteScanFailedImages {
		nonCompliantImages = append(nonCompliantImages, failedImages...)
	}

	if err := util.WriteImagesPipe(cfg.ctx, util.ScanErasePath, nonCompliantImages); err != nil {
		cfg.log.Error(err, "unable to write non-compliant images to scan erase pipe")
		return err
	}

	if cfg.reportMetrics {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		exporter, reader, provider := metrics.ConfigureMetrics(ctx, cfg.log, os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
		otel.SetMeterProvider(provider)

		if err := metrics.RecordMetricsScanner(ctx, otel.GetMeterProvider(), len(nonCompliantImages)); err != nil {
			cfg.log.Error(err, "error recording metrics")
			return err
		}

		metrics.ExportMetrics(cfg.log, exporter, reader)
	}
	return nil
}

func (cfg *config) Finish() error {
	defer func() { _ = cfg.completion.Close() }()

	// Finish is called by out-of-tree scanners, so it keeps its signature and
	// waits indefinitely, the same way WriteScanErasePipe does.
	data, err := cfg.completion.Await(context.Background())
	if err != nil {
		cfg.log.Error(err, "failed to read pipe", "pipeName", util.EraseCompleteScanPath)
		return err
	}

	if string(data) != util.EraseCompleteMessage {
		cfg.log.Info("garbage in pipe", "pipeName", util.EraseCompleteScanPath, "in_pipe", string(data))
		return err
	}

	cfg.log.Info("scanning complete, exiting")
	return nil
}

// provide custom context.
func WithContext(ctx context.Context) ConfigFunc {
	return func(cfg *config) {
		cfg.ctx = ctx
	}
}

// sets deleteScanFailedImages flag.
func WithDeleteScanFailedImages(deleteScanFailedImages bool) ConfigFunc {
	return func(cfg *config) {
		cfg.deleteScanFailedImages = deleteScanFailedImages
	}
}

// sets deleteEOLimages flag.
func WithDeleteEOLImages(deleteEOLImages bool) ConfigFunc {
	return func(cfg *config) {
		cfg.deleteEOLImages = deleteEOLImages
	}
}

// provide custom logger.
func WithLogger(log logr.Logger) ConfigFunc {
	return func(cfg *config) {
		cfg.log = log
	}
}

// sets boolean for recording metrics.
func WithMetrics(reportMetrics bool) ConfigFunc {
	return func(cfg *config) {
		cfg.reportMetrics = reportMetrics
	}
}
