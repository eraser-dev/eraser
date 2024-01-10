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

func TestUpdateImageList(t *testing.T) {
	imglistChangeFeat := features.New("Updating the Imagelist should trigger an ImageJob").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Deploy 2 deployments with different images (nginx, redis)
			nginxDep := util.NewDeployment(cfg.Namespace(), util.Nginx, 2, map[string]string{"app": util.Nginx}, corev1.Container{Image: util.Nginx, Name: util.Nginx})
			if err := cfg.Client().Resources().Create(ctx, nginxDep); err != nil {
				t.Error("Failed to create the nginx dep", err)
			}

			util.NewDeployment(cfg.Namespace(), util.Redis, 2, map[string]string{"app": util.Redis}, corev1.Container{Image: util.Redis, Name: util.Redis})
			err := cfg.Client().Resources().Create(ctx, util.NewDeployment(cfg.Namespace(), util.Redis, 2, map[string]string{"app": util.Redis}, corev1.Container{Image: util.Redis, Name: util.Redis}))
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Assess("Deployments successfully deployed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := cfg.NewClient()
			if err != nil {
				t.Error("Failed to create new client", err)
			}

			nginxDep := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: util.Nginx, Namespace: cfg.Namespace()},
			}

			if err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&nginxDep, appsv1.DeploymentAvailable, corev1.ConditionTrue),
				wait.WithTimeout(util.Timeout)); err != nil {
				t.Fatal("nginx deployment not found", err)
			}
			ctx = context.WithValue(ctx, util.Nginx, &nginxDep)

			redisDep := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: util.Redis, Namespace: cfg.Namespace()},
			}
			if err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&redisDep, appsv1.DeploymentAvailable, corev1.ConditionTrue),
				wait.WithTimeout(util.Timeout)); err != nil {
				t.Fatal("redis deployment not found", err)
			}
			ctx = context.WithValue(ctx, util.Redis, &redisDep)

			return ctx
		}).
		Assess("Remove deployments so the images aren't running", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Here we remove the redis and nginx deployments
			var redisPods corev1.PodList
			if err := cfg.Client().Resources().List(ctx, &redisPods, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{"app": util.Redis}).String()
			}); err != nil {
				t.Fatal(err)
			}
			if len(redisPods.Items) != 2 {
				t.Fatal("missing pods in redis deployment")
			}

			var nginxPods corev1.PodList
			if err := cfg.Client().Resources().List(ctx, &nginxPods, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{"app": util.Nginx}).String()
			}); err != nil {
				t.Fatal(err)
			}
			if len(nginxPods.Items) != 2 {
				t.Fatal("missing pods in nginx deployment")
			}

			err := cfg.Client().Resources().Delete(ctx, ctx.Value(util.Redis).(*appsv1.Deployment))
			if err != nil {
				t.Fatal(err)
			}
			err = cfg.Client().Resources().Delete(ctx, ctx.Value(util.Nginx).(*appsv1.Deployment))
			if err != nil {
				t.Fatal(err)
			}

			for _, nodeName := range util.GetClusterNodes(t) {
				err := wait.For(util.ContainerNotPresentOnNode(nodeName, util.Redis), wait.WithTimeout(util.Timeout))
				if err != nil {
					// Let's not mark this as an error
					// We only have this to prevent race conditions with the eraser spinning up
					t.Logf("error while waiting for deployment deletion: %v", err)
				}
			}
			for _, nodeName := range util.GetClusterNodes(t) {
				err := wait.For(util.ContainerNotPresentOnNode(nodeName, util.Nginx), wait.WithTimeout(util.Timeout))
				if err != nil {
					// Let's not mark this as an error
					// We only have this to prevent race conditions with the eraser spinning up
					t.Logf("error while waiting for deployment deletion: %v", err)
				}
			}
			return ctx
		}).
		Assess("Deploy imagelist to remove nginx", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// deploy imageJob config
			if err := util.DeployEraserConfig(cfg.KubeconfigFile(), cfg.Namespace(), util.EraserV1Alpha1ImagelistPath); err != nil {
				t.Error("Failed to deploy image list config", err)
			}

			ctxT, cancel := context.WithTimeout(ctx, util.Timeout)
			defer cancel()
			util.CheckImageRemoved(ctxT, t, util.GetClusterNodes(t), util.Nginx)

			return ctx
		}).
		Assess("Update imagelist to prune rest of images", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// deploy imageJob config
			if err := util.DeployEraserConfig(cfg.KubeconfigFile(), cfg.Namespace(), util.EraserV1Alpha1ImagelistUpdatedPath); err != nil {
				t.Error("Failed to deploy image list config", err)
			}

			ctxT, cancel := context.WithTimeout(ctx, util.Timeout)
			defer cancel()
			util.CheckImageRemoved(ctxT, t, util.GetClusterNodes(t), util.Redis)

			return ctx
		}).
		Assess("Get logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := util.GetPodLogs(t); err != nil {
				t.Error("error getting eraser pod logs", err)
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, imglistChangeFeat)
}
