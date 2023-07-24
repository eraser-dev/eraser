//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/eraser-dev/eraser/test/e2e/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestExclusionList(t *testing.T) {
	excludedImageFeat := features.New("Verify Eraser will skip excluded images").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			podSelectorLabels := map[string]string{"app": util.Nginx}
			nginxDep := util.NewDeployment(cfg.Namespace(), util.Nginx, 2, podSelectorLabels, corev1.Container{Image: util.Nginx, Name: util.Nginx})
			if err := cfg.Client().Resources().Create(ctx, nginxDep); err != nil {
				t.Error("Failed to create the dep", err)
			}
			if err := util.DeleteImageListsAndJobs(cfg.KubeconfigFile()); err != nil {
				t.Error("Failed to clean eraser obejcts ", err)
			}

			// create excluded configmap and add docker.io/library/*
			excluded := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "excluded",
					Namespace: cfg.Namespace(),
					Labels:    map[string]string{"eraser.sh/exclude.list": "true"},
				},
				Data: map[string]string{"test.json": "{\"excluded\": [\"docker.io/library/*\"]}"},
			}
			if err := cfg.Client().Resources().Create(ctx, &excluded); err != nil {
				t.Error("failed to create excluded configmap", err)
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
		Assess("Check image remains in all nodes", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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
					t.Logf("error while waiting for deployment deletion: %v", err)
				}
			}

			// create imagelist to trigger deletion
			if err := util.DeployEraserConfig(cfg.KubeconfigFile(), cfg.Namespace(), util.EraserV1ImagelistPath); err != nil {
				t.Error("Failed to deploy image list config", err)
			}

			_, cancel := context.WithTimeout(ctx, util.Timeout)
			defer cancel()
			// since docker.io/library/* was excluded, nginx should still exist following deletion
			util.CheckImagesExist(t, util.GetClusterNodes(t), util.Nginx)

			return ctx
		}).
		Assess("Get logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := util.GetPodLogs(t); err != nil {
				t.Error("error getting eraser pod logs", err)
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, excludedImageFeat)
}
