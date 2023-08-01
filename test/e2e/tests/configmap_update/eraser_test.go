//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/eraser-dev/eraser/test/e2e/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	numPods       = 3
	configKey     = "controller_manager_config.yaml"
	configmapName = "eraser-manager-config"
)

var ()

func TestConfigmapUpdate(t *testing.T) {
	metrics := features.New("Updating the remover image in the configmap should cause the manager to deploy using the new image").
		Assess("Update configmap, change remover image to busybox", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := cfg.NewClient()
			if err != nil {
				t.Error("Failed to create new client", err)
			}

			configMap := corev1.ConfigMap{}
			err = client.Resources().Get(ctx, configmapName, util.TestNamespace, &configMap)
			if err != nil {
				t.Error("Unable to get configmap", err)
			}

			bbSplit := strings.Split(util.BusyboxImage, ":")
			bbRepo := bbSplit[0]
			bbTag := bbSplit[1]

			cmString := fmt.Sprintf(`---
apiVersion: eraser.sh/v1alpha2
kind: EraserConfig
components:
  remover:
    image:
      repo: %s
      tag: %s
    `, bbRepo, bbTag)

			configMap.Data[configKey] = cmString
			err = client.Resources().Update(ctx, &configMap)
			if err != nil {
				t.Error("unable to update configmap", err)
			}

			return ctx
		}).
		Assess("Deploy Imagelist", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// deploy imagelist config
			if err := util.DeployEraserConfig(cfg.KubeconfigFile(), cfg.Namespace(), util.ImagelistAlpinePath); err != nil {
				t.Error("Failed to deploy image list config", err)
			}

			return ctx
		}).
		Assess("Check eraser pods for change in configuration", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Error("Failed to create new client", err)
			}

			err = wait.For(
				util.NumPodsPresentForLabel(ctx, c, numPods, util.ImageJobTypeLabelKey+"="+util.ManualLabel),
				wait.WithTimeout(time.Minute*2),
				wait.WithInterval(time.Millisecond*500),
			)
			if err != nil {
				t.Fatal(err)
			}

			var ls corev1.PodList
			err = c.Resources().List(ctx, &ls, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{util.ImageJobTypeLabelKey: util.ManualLabel}).String()
			})
			if err != nil {
				t.Errorf("could not list pods: %v", err)
			}

			for i := range ls.Items {
				// there will only be the remover container in an imagelist deployment
				container := ls.Items[i].Spec.Containers[0]
				image := container.Image
				if image != util.BusyboxImage {
					t.Errorf("pod %s has image %s, should be %s", ls.Items[i].GetName(), image, util.BusyboxImage)
				}
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
