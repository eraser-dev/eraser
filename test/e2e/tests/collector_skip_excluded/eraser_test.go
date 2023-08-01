//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/eraser-dev/eraser/test/e2e/util"

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
				o.LabelSelector = labels.SelectorFromSet(map[string]string{util.ImageJobTypeLabelKey: util.CollectorLabel}).String()
			})
			if err != nil {
				t.Errorf("could not list pods: %v", err)
			}

			for _, pod := range ls.Items {
				err = wait.For(conditions.New(c.Resources()).PodPhaseMatch(&pod, corev1.PodSucceeded), wait.WithTimeout(time.Minute*3))
				if err != nil {
					t.Log("collector pod unsuccessful", pod.Name)
				}
			}

			return ctx
		}).
		Assess("Alpine image is not removed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			_, cancel := context.WithTimeout(ctx, util.Timeout)
			defer cancel()
			util.CheckImagesExist(t, util.GetClusterNodes(t), util.VulnerableImage)

			return ctx
		}).
		Assess("Non-vulnerable image is not removed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			_, cancel := context.WithTimeout(ctx, util.Timeout)
			defer cancel()
			util.CheckImagesExist(t, util.GetClusterNodes(t), util.NonVulnerableImage)

			return ctx
		}).
		Assess("Get logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := util.GetPodLogs(t); err != nil {
				t.Error("error getting eraser pod logs", err)
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, collectorExcluded)
}
