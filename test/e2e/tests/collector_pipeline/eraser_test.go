//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/eraser-dev/eraser/test/e2e/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestCollectScanErasePipeline(t *testing.T) {
	collectScanErasePipelineFeat := features.New("Collector pods should run automatically, trigger the scanner, then the eraser pods. Manifest deployment test.").
		Assess("Vulnerable and EOL images are successfully deleted from all nodes", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ctxT, cancel := context.WithTimeout(ctx, util.Timeout)
			defer cancel()
			util.CheckImageRemoved(ctxT, t, util.GetClusterNodes(t), util.Alpine)

			return ctx
		}).
		Assess("Pods from imagejobs are cleaned up", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Fatal("Failed to create new client", err)
			}

			var ls corev1.PodList
			err = c.Resources().List(ctx, &ls, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{util.ImageJobTypeLabelKey: util.CollectorLabel}).String()
			})
			if err != nil {
				t.Errorf("could not list pods: %v", err)
			}

			err = wait.For(conditions.New(c.Resources()).ResourcesDeleted(&ls), wait.WithTimeout(util.Timeout))
			if err != nil {
				t.Errorf("error waiting for pods to be deleted: %v", err)
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, collectScanErasePipelineFeat)
}
