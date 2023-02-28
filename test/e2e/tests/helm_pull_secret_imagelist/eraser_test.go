//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/eraser/test/e2e/util"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	collectorLabel = "name=eraser"

	expectedPods = 4
)

func TestHelmPullSecret(t *testing.T) {
	pullSecretsPropagated := features.New("Image Pull Secrets").
		Assess("All pods should have the correct pull secret", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Fatal("Failed to create new client", err)
			}

			imgList := &eraserv1alpha1.ImageList{
				ObjectMeta: metav1.ObjectMeta{Name: util.Prune},
				Spec: eraserv1alpha1.ImageListSpec{
					Images: []string{"*"},
				},
			}
			if err := cfg.Client().Resources().Create(ctx, imgList); err != nil {
				t.Fatal(err)
			}

			err = wait.For(
				util.NumPodsPresentForLabel(ctx, c, 3, collectorLabel),
				wait.WithTimeout(time.Minute*2),
				wait.WithInterval(time.Millisecond*500),
			)
			if err != nil {
				t.Fatal(err)
			}

			var ls corev1.PodList
			err = c.Resources().List(ctx, &ls, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{"name": "eraser"}).String()
			})
			if err != nil {
				t.Errorf("could not list pods: %v", err)
			}

			var ls2 corev1.PodList
			err = c.Resources().List(ctx, &ls2, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{"control-plane": "controller-manager"}).String()
			})

			items := append(ls.Items, ls2.Items...)
			if len(items) != expectedPods {
				t.Errorf("incorrect number of pods for eraser deployment. should be %d but was %d", expectedPods, len(items))
			}

			for _, pod := range items {
				found := false
				for _, secret := range pod.Spec.ImagePullSecrets {
					if secret.Name == util.ImagePullSecret {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("pod %s does not have secret set", pod.ObjectMeta.Name)
				}
			}

			return ctx
		}).
		Assess("Get logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := util.GetPodLogs(ctx, cfg, t, true); err != nil {
				t.Error("error getting collector pod logs", err)
			}

			if err := util.GetManagerLogs(ctx, cfg, t); err != nil {
				t.Error("error getting manager logs", err)
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, pullSecretsPropagated)
}
