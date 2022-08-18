//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/eraser/test/e2e/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestImageListTriggersEraserImageJob(t *testing.T) {
	rmImageFeat := features.New("An ImageList should trigger an eraser ImageJob").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			podSelectorLabels := map[string]string{"app": util.Nginx}
			nginxDep := util.NewDeployment(cfg.Namespace(), util.Nginx, 2, podSelectorLabels, corev1.Container{Image: util.Nginx, Name: util.Nginx})
			if err := cfg.Client().Resources().Create(ctx, nginxDep); err != nil {
				t.Error("Failed to create the dep", err)
			}
			if err := util.DeleteImageListsAndJobs(cfg.KubeconfigFile()); err != nil {
				t.Error("Failed to clean eraser obejcts ", err)
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
				wait.WithTimeout(time.Minute*3)); err != nil {
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
				err := wait.For(util.ContainerNotPresentOnNode(nodeName, util.Nginx), wait.WithTimeout(time.Minute*2))
				if err != nil {
					// Let's not mark this as an error
					// We only have this to prevent race conditions with the eraser spinning up
					t.Logf("error while waiting for deployment deletion: %v", err)
				}
			}

			// deploy imageJob config
			if err := util.DeployEraserConfig(cfg.KubeconfigFile(), util.EraserNamespace, "../../test-data", "eraser_v1alpha1_imagelist.yaml"); err != nil {
				t.Error("Failed to deploy image list config", err)
			}

			ctxT, cancel := context.WithTimeout(ctx, time.Minute*3)
			defer cancel()
			util.CheckImageRemoved(ctxT, t, util.GetClusterNodes(t), util.Nginx)

			// get logs after job completion
			job, err := util.GetImageJob(ctx, cfg)
			if err != nil {
				t.Error(err)
			}

			err = wait.For(conditions.New(client.Resources()).JobCompleted(job), wait.WithTimeout(time.Minute*2))
			if err != nil {
				t.Error("error waiting for imagejob completion")
			}

			eraserLogs, err := util.GetEraserLogs(ctx, cfg)
			if err != nil {
				t.Error("error getting eraser logs", err)
			}
			t.Log("eraser logs\n", eraserLogs)

			managerLogs, err := util.GetManagerLogs(ctx, cfg)
			if err != nil {
				t.Error("error getting manager logs", err)
			}
			t.Log("manager logs\n", managerLogs)

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, rmImageFeat)
}
