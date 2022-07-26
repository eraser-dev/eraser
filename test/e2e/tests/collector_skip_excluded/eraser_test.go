//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/test/e2e/util"

	"strings"

	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestCollectorExcluded(t *testing.T) {
	collectorExcluded := features.New("ImageCollector should remove excluded images from imagecollector-shared").
		Assess("Alpine image is not removed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ctxT, cancel := context.WithTimeout(ctx, time.Minute*5)
			defer cancel()
			util.CheckImageExists(ctxT, t, util.GetClusterNodes(t), util.VulnerableImage)

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, collectorExcluded)
}
