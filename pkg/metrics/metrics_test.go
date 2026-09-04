package metrics

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestConfigureMetrics(t *testing.T) {
	exporter, reader, provider := ConfigureMetrics(context.Background(), logr.Discard(), "otel-collector:4318")
	if exporter == nil {
		t.Fatal("unable to configure exporter")
	}
	if reader == nil {
		t.Fatal("unable to configure exporter")
	}
	if provider == nil {
		t.Fatal("unable to configure exporter")
	}

	otel.SetMeterProvider(provider)
}

// capturingExporter records the context Export was handed, so a regression back
// to context.Background() inside ExportMetrics is visible rather than silent.
type capturingExporter struct {
	got context.Context
}

func (e *capturingExporter) Temporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (e *capturingExporter) Aggregation(sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.AggregationDefault{}
}

func (e *capturingExporter) Export(ctx context.Context, _ *metricdata.ResourceMetrics) error {
	e.got = ctx
	return nil
}

func (e *capturingExporter) ForceFlush(context.Context) error { return nil }
func (e *capturingExporter) Shutdown(context.Context) error   { return nil }

func TestExportMetricsUsesTheCallersContext(t *testing.T) {
	type marker struct{}

	exporter := &capturingExporter{}
	reader := sdkmetric.NewManualReader()
	// Registering the reader is what makes Collect succeed, so the export half
	// is actually reached.
	sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	ctx := context.WithValue(context.Background(), marker{}, "caller")
	ExportMetrics(ctx, logr.Discard(), exporter, reader)

	require.NotNil(t, exporter.got, "Export was never called, so nothing was verified")
	assert.Equal(t, "caller", exporter.got.Value(marker{}))
}

func TestExportMetricsCarriesCancellationToTheExporter(t *testing.T) {
	exporter := &capturingExporter{}
	reader := sdkmetric.NewManualReader()
	sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ExportMetrics(ctx, logr.Discard(), exporter, reader)

	// Collect may reject the canceled context and return before the export. That
	// is fine -- what must not happen is an export running under a context that
	// knows nothing about the shutdown.
	if exporter.got != nil {
		assert.ErrorIs(t, exporter.got.Err(), context.Canceled)
	}
}

func TestRecordMetrics(t *testing.T) {
	if err := RecordMetricsRemover(context.Background(), otel.GetMeterProvider(), 1); err != nil {
		t.Fatal("could not record eraser metrics")
	}

	if err := RecordMetricsScanner(context.Background(), otel.GetMeterProvider(), 1); err != nil {
		t.Fatal("could not record scanner metrics")
	}

	if err := RecordMetricsController(context.Background(), otel.GetMeterProvider(), 1.0, 1, 1); err != nil {
		t.Fatal("could not record scanner metrics")
	}
}

func TestMeterCreatesInstrument(t *testing.T) {
	testCases := []struct {
		name string
		fn   func(*testing.T, metric.Meter)
	}{
		{
			name: "AsyncInt64Count",
			fn: func(t *testing.T, m metric.Meter) {
				ctr, err := m.Int64Counter(ImagesRemovedCounter)
				assert.NoError(t, err)
				ctr.Add(context.Background(), 1)
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			rdr := sdkmetric.NewManualReader()
			m := sdkmetric.NewMeterProvider(sdkmetric.WithReader(rdr)).Meter("eraser")

			tt.fn(t, m)

			var rm metricdata.ResourceMetrics
			err := rdr.Collect(context.Background(), &rm)
			assert.NoError(t, err)

			require.Len(t, rm.ScopeMetrics, 1)
			sm := rm.ScopeMetrics[0]
			require.Len(t, sm.Metrics, 1)
			got := sm.Metrics[0]

			if got.Name != ImagesRemovedCounter {
				t.Error("ImagesRemovedCounter not created")
			}
		})
	}
}
