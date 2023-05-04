package metrics

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
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

	global.SetMeterProvider(provider)
}

func TestRecordMetrics(t *testing.T) {
	if err := RecordMetricsEraser(context.Background(), global.MeterProvider(), 1); err != nil {
		t.Fatal("could not record eraser metrics")
	}

	if err := RecordMetricsScanner(context.Background(), global.MeterProvider(), 1); err != nil {
		t.Fatal("could not record scanner metrics")
	}

	if err := RecordMetricsController(context.Background(), global.MeterProvider(), 1.0, 1, 1); err != nil {
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

			metrics := metricdata.ResourceMetrics{}
			err := rdr.Collect(context.Background(), &metrics)
			assert.NoError(t, err)

			require.Len(t, metrics.ScopeMetrics, 1)
			sm := metrics.ScopeMetrics[0]
			require.Len(t, sm.Metrics, 1)
			got := sm.Metrics[0]

			if got.Name != ImagesRemovedCounter {
				t.Error("ImagesRemovedCounter not created")
			}
		})
	}
}
