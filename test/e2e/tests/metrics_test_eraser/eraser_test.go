//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"regexp"
	"strconv"
	"testing"

	"github.com/Azure/eraser/test/e2e/util"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	expectedImagesRemoved = 3
)

func TestMetrics(t *testing.T) {
	metrics := features.New("Images_removed_run_total metric should report 1").
		Assess("Alpine image is removed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// deploy imagelist config
			if err := util.DeployEraserConfig(cfg.KubeconfigFile(), util.TestNamespace, "../../test-data", "imagelist_alpine.yaml"); err != nil {
				t.Error("Failed to deploy image list config", err)
			}

			ctxT, cancel := context.WithTimeout(ctx, util.Timeout)
			defer cancel()
			util.CheckImageRemoved(ctxT, t, util.GetClusterNodes(t), util.VulnerableImage)

			return ctx
		}).
		Assess("Check images_removed_run_total metric", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if _, err := util.KubectlCurlPod(cfg.KubeconfigFile(), util.TestNamespace); err != nil {
				t.Error(err, "error running curl pod")
			}

			if _, err := util.KubectlWait(cfg.KubeconfigFile(), "temp", util.TestNamespace); err != nil {
				t.Error(err, "error waiting for temp curl pod")
			}

			output, err := util.KubectlExecCurl(cfg.KubeconfigFile(), "temp", "http://otel-collector/metrics", util.TestNamespace)
			if err != nil {
				t.Error(err, "error with otlp curl request")
			}

			r := regexp.MustCompile(`images_removed_run_total{job="eraser",node_name=".+"} (\d+)`)
			results := r.FindAllStringSubmatch(output, -1)

			totalRemoved := 0
			for i := range results {
				val, _ := strconv.Atoi(results[i][1])
				totalRemoved += val
			}

			if totalRemoved != expectedImagesRemoved {
				t.Error("images_removed_run_total incorrect, expected ", expectedImagesRemoved, "got", totalRemoved)
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, metrics)
}
