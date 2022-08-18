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

func TestDisableScanner(t *testing.T) {
	disableScanFeat := features.New("Scanner can be disabled").
		// non-vulnerable image should be deleted from all nodes when scanner is disabled and we prune with collector
		Assess("Non-vulnerable image successfully deleted from all nodes", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ctxT, cancel := context.WithTimeout(ctx, time.Minute*5)
			defer cancel()
			util.CheckImageRemoved(ctxT, t, util.GetClusterNodes(t), util.NonVulnerableImage)

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

			// get logs after job completion
			job, err := util.GetImageJob(ctx, cfg)
			if err != nil {
				t.Error(err)
			}

			err = wait.For(conditions.New(client.Resources()).JobCompleted(job), wait.WithTimeout(time.Minute*2))
			if err != nil {
				t.Error("error waiting for imagejob completion")
			}

			for _, pod := range ls.Items {
				t.Log("pod name", pod.Name)
				var output string

				output, err = util.KubectlLogs(cfg.KubeconfigFile(), pod.Name, "collector", util.EraserNamespace)
				if err != nil {
					t.Error("could not get collector container output", err)
				}
				t.Log("collector output\n", output)

				output, err := util.KubectlLogs(cfg.KubeconfigFile(), pod.Name, "eraser", util.EraserNamespace)
				if err != nil {
					t.Error("could not get eraser container output", err)
				}
				t.Log("eraser output\n", output)
			}

			managerLogs, err := util.GetManagerLogs(ctx, cfg)
			if err != nil {
				t.Error("error getting manager logs", err)
			}

			t.Log("manager logs\n", managerLogs)

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, disableScanFeat)
}
