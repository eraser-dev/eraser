package metrics

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	metric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/unit"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
)

const (
	ImagesRemovedCounter     = "images_removed_run_total"
	ImagesRemovedDescription = "total images removed"
)

func ConfigureMetrics(ctx context.Context, log logr.Logger, endpoint string) (sdkmetric.Exporter, sdkmetric.Reader, *sdkmetric.MeterProvider) {
	exporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithInsecure(), otlpmetrichttp.WithEndpoint(endpoint))
	if err != nil {
		log.Error(err, "error initializing exporter")
		return nil, nil, nil
	}

	reader := sdkmetric.NewPeriodicReader(exporter)

	durationInstrument := sdkmetric.Instrument{
		Name:  "imagejob_duration_run_seconds",
		Scope: instrumentation.Scope{Name: "eraser"},
	}

	durationStream := sdkmetric.Stream{
		Name: "imagejob_duration_run_seconds",
		Unit: unit.Unit("s"),
		Aggregation: aggregation.ExplicitBucketHistogram{
			Boundaries: []float64{0, 10, 20, 30, 40, 50, 60},
		},
	}

	histogramView := sdkmetric.NewView(durationInstrument, durationStream)

	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader), sdkmetric.WithView(histogramView))

	return exporter, reader, provider
}

func ExportMetrics(log logr.Logger, exporter sdkmetric.Exporter, reader sdkmetric.Reader) {
	ctxB := context.Background()

	m, err := reader.Collect(ctxB)
	if err != nil {
		log.Error(err, "failed to collect metrics")
		return
	}
	if err := exporter.Export(ctxB, m); err != nil {
		log.Error(err, "failed to export metrics")
	}
}

func RecordMetricsRemover(ctx context.Context, p metric.MeterProvider, totalRemoved int64) error {
	counter, err := p.Meter("eraser").SyncInt64().Counter(ImagesRemovedCounter, instrument.WithDescription(ImagesRemovedDescription), instrument.WithUnit("1"))
	if err != nil {
		return err
	}

	counter.Add(ctx, totalRemoved, attribute.String("node name", os.Getenv("NODE_NAME")))
	return nil
}

func RecordMetricsScanner(ctx context.Context, p metric.MeterProvider, totalVulnerable int) error {
	counter, err := p.Meter("eraser").SyncInt64().Counter("vulnerable_images_run_total", instrument.WithDescription("total vulnerable images"), instrument.WithUnit("1"))
	if err != nil {
		return err
	}

	counter.Add(ctx, int64(totalVulnerable), attribute.String("node name", os.Getenv("NODE_NAME")))
	return nil
}

func RecordMetricsController(ctx context.Context, p metric.MeterProvider, jobDuration float64, podsCompleted int64, podsFailed int64) error {
	duration, err := p.Meter("eraser").SyncFloat64().Histogram("imagejob_duration_run_seconds", instrument.WithDescription("duration of imagejob"), instrument.WithUnit(unit.Unit("s")))
	if err != nil {
		return err
	}
	duration.Record(ctx, jobDuration)

	completed, err := p.Meter("eraser").SyncInt64().Counter("pods_completed_run_total", instrument.WithDescription("total pods completed"), instrument.WithUnit("1"))
	if err != nil {
		return err
	}
	completed.Add(ctx, podsCompleted)

	failed, err := p.Meter("eraser").SyncInt64().Counter("pods_failed_run_total", instrument.WithDescription("total pods failed"), instrument.WithUnit("1"))
	if err != nil {
		return err
	}
	failed.Add(ctx, podsFailed)

	jobTotal, err := p.Meter("eraser").SyncInt64().Counter("imagejob_run_total", instrument.WithDescription("total number of imagejobs completed"), instrument.WithUnit("1"))
	if err != nil {
		return err
	}
	jobTotal.Add(ctx, 1)

	return nil
}
