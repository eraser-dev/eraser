//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/eraser-dev/eraser/test/e2e/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	restartTimeout = time.Minute
)

func TestImageListTriggersRemoverImageJob(t *testing.T) {
	rmImageFeat := features.New("An ImageList should trigger a remover ImageJob").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			podSelectorLabels := map[string]string{"app": util.Nginx}
			nginxDep := util.NewDeployment(cfg.Namespace(), util.Nginx, 2, podSelectorLabels, corev1.Container{Image: util.Nginx, Name: util.Nginx})
			if err := cfg.Client().Resources().Create(ctx, nginxDep); err != nil {
				t.Error("Failed to create the dep", err)
			}

			return ctx
		}).
		Assess("deployment successfully deployed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := cfg.NewClient()
			if err != nil {
				t.Error("Failed to create new client", err)
			}

			resultDeployment := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: util.Nginx, Namespace: cfg.Namespace()},
			}

			if err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&resultDeployment, appsv1.DeploymentAvailable, corev1.ConditionTrue),
				wait.WithTimeout(util.Timeout)); err != nil {
				t.Error("deployment not found", err)
			}

			return context.WithValue(ctx, util.Nginx, &resultDeployment)
		}).
		Assess("Images successfully deleted from all nodes", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// delete deployment
			client, err := cfg.NewClient()
			if err != nil {
				t.Error("Failed to create new client", err)
			}

			var pods corev1.PodList
			err = client.Resources().List(ctx, &pods, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(labels.Set{"app": util.Nginx}).String()
			})
			if err != nil {
				t.Fatal(err)
			}

			dep := ctx.Value(util.Nginx).(*appsv1.Deployment)
			if err := client.Resources().Delete(ctx, dep); err != nil {
				t.Error("Failed to delete the dep", err)
			}

			for _, nodeName := range util.GetClusterNodes(t) {
				err := wait.For(util.ContainerNotPresentOnNode(nodeName, util.Nginx), wait.WithTimeout(util.Timeout))
				if err != nil {
					// Let's not mark this as an error
					// We only have this to prevent race conditions with the eraser spinning up
					t.Logf("error while waiting for deployment deletion: %v", err)
				}
			}

			// deploy imageJob config
			if err := util.DeployEraserConfig(cfg.KubeconfigFile(), cfg.Namespace(), util.EraserV1Alpha1ImagelistPath); err != nil {
				t.Error("Failed to deploy image list config", err)
			}

			podNames := []string{}
			// get eraser pod name
			err = wait.For(func() (bool, error) {
				l := corev1.PodList{}
				err = client.Resources().List(ctx, &l, resources.WithLabelSelector(util.ImageJobTypeLabelKey+"="+util.ManualLabel))
				if err != nil {
					return false, err
				}

				if len(l.Items) != 3 {
					return false, nil
				}

				for _, pod := range l.Items {
					podNames = append(podNames, pod.ObjectMeta.Name)
				}
				return true, nil
			}, wait.WithTimeout(time.Minute*2), wait.WithInterval(time.Millisecond*500))
			if err != nil {
				t.Fatal(err)
			}

			// wait for those specific pods to no longer exist, so that when we
			// check later for an accidental redeployment, we are sure it is
			// actually a new deployment.
			err = wait.For(func() (bool, error) {
				var l corev1.PodList
				err = client.Resources().List(ctx, &l, resources.WithLabelSelector(util.ImageJobTypeLabelKey+"="+util.ManualLabel))
				if err != nil {
					return false, err
				}

				if len(l.Items) == 0 {
					return true, nil
				}

				for _, name := range podNames {
					for _, pod := range l.Items {
						if name == pod.ObjectMeta.Name {
							return false, nil
						}
					}
				}

				return true, nil
			}, wait.WithTimeout(util.Timeout), wait.WithInterval(time.Millisecond*500))
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("initial eraser deployment cleaned up")

			ctxT, cancel := context.WithTimeout(ctx, time.Minute*3)
			defer cancel()
			util.CheckImageRemoved(ctxT, t, util.GetClusterNodes(t), util.Nginx)

			return ctx
		}).
		Assess("Eraser job was not restarted", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// until a timeout is reached, make sure there are no pods matching
			// the selector eraser.sh/type=manual
			client := cfg.Client()
			ctxT2, cancel := context.WithTimeout(ctx, restartTimeout)
			defer cancel()
			util.CheckDeploymentCleanedUp(ctxT2, t, client)

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, rmImageFeat)
}
