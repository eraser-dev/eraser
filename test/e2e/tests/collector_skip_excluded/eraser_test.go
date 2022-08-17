//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/eraser/test/e2e/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestCollectorExcluded(t *testing.T) {
	collectorExcluded := features.New("ImageCollector should not remove excluded images from imagecollector-shared").
		Assess("Alpine image is not removed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ctxT, cancel := context.WithTimeout(ctx, time.Minute*5)
			defer cancel()
			util.CheckImagesExist(ctxT, t, util.GetClusterNodes(t), util.VulnerableImage)

			return ctx
		}).
		Assess("Non-vulnerable image is not removed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ctxT, cancel := context.WithTimeout(ctx, time.Minute*5)
			defer cancel()
			util.CheckImagesExist(ctxT, t, util.GetClusterNodes(t), util.NonVulnerableImage)

			return ctx
		}).
		Assess("Pods from imagejobs are cleaned up", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Fatal("Failed to create new client", err)
			}

			var ls corev1.PodList
			err = c.Resources().List(ctx, &ls, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{"name": "collector"}).String()
			})
			if err != nil {
				t.Errorf("could not list pods: %v", err)
			}

			err = wait.For(conditions.New(c.Resources()).ResourcesDeleted(&ls), wait.WithTimeout(time.Minute))
			if err != nil {
				t.Errorf("error waiting for pods to be deleted: %v", err)
			}

			managerLogs, err := util.GetManagerLogs(cfg, ctx)
			if err != nil {
				t.Error("error getting manager logs", err)
			}

			t.Log("manager logs\n", managerLogs)

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, collectorExcluded)
}
