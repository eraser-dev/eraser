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
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const (
	ImagesRemovedCounter     = "ImagesRemoved"
	ImagesRemovedDescription = "total images removed"
)

func ConfigureMetrics(ctx context.Context, log logr.Logger, endpoint string) (sdkmetric.Exporter, sdkmetric.Reader, *sdkmetric.MeterProvider) {
	exporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithInsecure(), otlpmetrichttp.WithEndpoint(endpoint))
	if err != nil {
		log.Error(err, "error initializing exporter")
		return nil, nil, nil
	}

	reader := sdkmetric.NewPeriodicReader(exporter)
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	return exporter, reader, provider
}

func ExportMetrics(log logr.Logger, exporter sdkmetric.Exporter, reader sdkmetric.Reader, provider *sdkmetric.MeterProvider) {
	ctxB := context.Background()

	m, err := reader.Collect(ctxB)
	if err != nil {
		log.Error(err, "failed to collect metrics")
		return
	}
	if err := exporter.Export(ctxB, m); err != nil {
		log.Error(err, "failed to export metrics")
	}
	if err := provider.Shutdown(ctxB); err != nil {
		log.Error(err, "error during metric shutdown")
	}
}

func RecordMetricsEraser(ctx context.Context, p metric.MeterProvider, totalRemoved int64) error {
	counter, err := p.Meter("eraser").SyncInt64().Counter(ImagesRemovedCounter, instrument.WithDescription(ImagesRemovedDescription), instrument.WithUnit("1"))
	if err != nil {
		return err
	}

	counter.Add(ctx, totalRemoved, attribute.String("node name", os.Getenv("NODE_NAME")))
	return nil
}

func RecordMetricsScanner(ctx context.Context, p metric.MeterProvider, totalVulnerable int) error {
	counter, err := p.Meter("eraser").SyncInt64().Counter("VulnerableImages", instrument.WithDescription("total vulnerable images"), instrument.WithUnit("1"))
	if err != nil {
		return err
	}

	counter.Add(ctx, int64(totalVulnerable), attribute.String("node name", os.Getenv("NODE_NAME")))
	return nil
}

func RecordMetricsController(ctx context.Context, p metric.MeterProvider, jobDuration float64, podsCompleted int64, podsFailed int64) error {
	duration, err := p.Meter("eraser").SyncFloat64().Histogram("ImageJobCollectorDuration", instrument.WithDescription("duration of collector imagejob"), instrument.WithUnit(unit.Milliseconds))
	if err != nil {
		return err
	}
	duration.Record(ctx, jobDuration)

	completed, err := p.Meter("eraser").SyncInt64().Counter("PodsCompleted", instrument.WithDescription("total pods completed"), instrument.WithUnit("1"))
	if err != nil {
		return err
	}
	completed.Add(ctx, podsCompleted)

	failed, err := p.Meter("eraser").SyncInt64().Counter("PodsFailed", instrument.WithDescription("total pods failed"), instrument.WithUnit("1"))
	if err != nil {
		return err
	}
	failed.Add(ctx, podsFailed)

	jobTotal, err := p.Meter("eraser").SyncInt64().Counter("ImageJobCollectorTotal", instrument.WithDescription("total number of collector imagejobs completed"), instrument.WithUnit("1"))
	if err != nil {
		return err
	}
	jobTotal.Add(ctx, 1)

	return nil
}
