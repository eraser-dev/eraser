//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
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

func TestHelmPullSecret(t *testing.T) {
	pullSecretsPropagated := features.New("Image Pull Secrets").
		Assess("All pods should have the correct pull secret", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Fatal("Failed to create new client", err)
			}

			var lastReason string
			err = wait.For(
				func() (bool, error) {
					var collectors corev1.PodList
					if err := c.Resources().List(ctx, &collectors, func(o *metav1.ListOptions) {
						o.LabelSelector = labels.SelectorFromSet(map[string]string{util.ImageJobTypeLabelKey: util.CollectorLabel}).String()
					}); err != nil {
						return false, err
					}

					var managers corev1.PodList
					if err := c.Resources().List(ctx, &managers, func(o *metav1.ListOptions) {
						o.LabelSelector = labels.SelectorFromSet(map[string]string{"control-plane": "controller-manager"}).String()
					}); err != nil {
						return false, err
					}

					if len(collectors.Items) < 3 || len(managers.Items) < 1 {
						lastReason = fmt.Sprintf(
							"waiting for at least 3 collector pods and 1 manager pod; got collectors=%d managers=%d",
							len(collectors.Items),
							len(managers.Items),
						)
						return false, nil
					}

					for _, pod := range append(collectors.Items, managers.Items...) {
						found := false
						for _, secret := range pod.Spec.ImagePullSecrets {
							if secret.Name == util.ImagePullSecret {
								found = true
								break
							}
						}
						if !found {
							lastReason = fmt.Sprintf(
								"pod %s is missing image pull secret %s",
								pod.Name,
								util.ImagePullSecret,
							)
							return false, nil
						}
					}

					lastReason = ""
					return true, nil
				},
				wait.WithTimeout(time.Minute*2),
				wait.WithInterval(time.Millisecond*500),
			)
			if err != nil {
				if lastReason != "" {
					t.Fatalf("%v: %s", err, lastReason)
				}
				t.Fatal(err)
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

	util.Testenv.Test(t, pullSecretsPropagated)
}
