//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"regexp"
	"strconv"
	"testing"

	"github.com/eraser-dev/eraser/test/e2e/util"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	expectedImagesRemoved = 3
)

func TestMetricsWithScannerDisabled(t *testing.T) {
	metrics := features.New("Images_removed_run_total metric should report 1").
		Assess("Alpine image is removed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ctxT, cancel := context.WithTimeout(ctx, util.Timeout)
			defer cancel()
			util.CheckImageRemoved(ctxT, t, util.GetClusterNodes(t), util.VulnerableImage)

			return ctx
		}).
		Assess("Get logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := util.GetPodLogs(t); err != nil {
				t.Error("error getting eraser pod logs", err)
			}

			return ctx
		}).
		Assess("Check images_removed_run_total metric", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if _, err := util.KubectlCurlPod(cfg.KubeconfigFile(), cfg.Namespace()); err != nil {
				t.Error(err, "error running curl pod")
			}

			if _, err := util.KubectlWait(cfg.KubeconfigFile(), "temp", cfg.Namespace()); err != nil {
				t.Error(err, "error waiting for temp curl pod")
			}

			output, err := util.KubectlExecCurl(cfg.KubeconfigFile(), "temp", "http://otel-collector/metrics", cfg.Namespace())
			if err != nil {
				t.Error(err, "error with otlp curl request")
			}

			r := regexp.MustCompile(`images_removed_run_total{job="remover",node_name=".+"} (\d+)`)
			results := r.FindAllStringSubmatch(output, -1)

			totalRemoved := 0
			for i := range results {
				val, _ := strconv.Atoi(results[i][1])
				totalRemoved += val
			}

			if totalRemoved < expectedImagesRemoved {
				t.Error("images_removed_run_total incorrect, expected ", expectedImagesRemoved, "got", totalRemoved)
			}

			return ctx
		}).
		Assess("Get logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := util.GetPodLogs(t); err != nil {
				t.Error("error getting eraser pod logs", err)
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, metrics)
}
