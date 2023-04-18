//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/Azure/eraser/test/e2e/util"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestEnsureScannerFunctions(t *testing.T) {
	collectScanErasePipelineFeat := features.New("Collector pods should run automatically, trigger the scanner, then the eraser pods. Helm test.").
		Assess("Vulnerable and EOL images are successfully deleted from all nodes", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ctxT, cancel := context.WithTimeout(ctx, util.Timeout)
			defer cancel()
			util.CheckImageRemoved(ctxT, t, util.GetClusterNodes(t), util.Alpine)

			return ctx
		}).
		Assess("Get logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := util.GetManagerLogs(ctx, cfg, t); err != nil {
				t.Error("error getting manager logs", err)
			}

			if err := util.GetPodLogs(ctx, cfg, t, false); err != nil {
				t.Error("error getting manager logs", err)
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, collectScanErasePipelineFeat)
}
