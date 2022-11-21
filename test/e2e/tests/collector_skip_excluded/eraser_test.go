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
	collectorExcluded := features.New("ImageCollector should not remove excluded images").
		Assess("Collector pods completed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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

			for _, pod := range ls.Items {
				err = wait.For(conditions.New(c.Resources()).PodPhaseMatch(&pod, corev1.PodSucceeded), wait.WithTimeout(time.Minute*3))
				if err != nil {
					t.Error("collector pod unsuccessful", err, pod.Name)
				}
			}

			return ctx
		}).
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
		Assess("Get logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := util.GetPodLogs(ctx, cfg, t, false); err != nil {
				t.Error("error getting collector pod logs", err)
			}

			if err := util.GetManagerLogs(ctx, cfg, t); err != nil {
				t.Error("error getting manager logs", err)
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, collectorExcluded)
}
