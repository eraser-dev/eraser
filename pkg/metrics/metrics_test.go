package metrics

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	metric "go.opentelemetry.io/otel/metric"
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
