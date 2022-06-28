//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
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

func TestRemoveImagesFromAllNodes(t *testing.T) {
	const (
		nginx         = "nginx"
		nginxLatest   = "docker.io/library/nginx:latest"
		nginxAliasOne = "docker.io/library/nginx:one"
		nginxAliasTwo = "docker.io/library/nginx:two"
		redis         = "redis"
		caddy         = "caddy"

		prune               = "imagelist"
		skippedNodeName     = "eraser-e2e-test-worker"
		skippedNodeSelector = "kubernetes.io/hostname=eraser-e2e-test-worker"
		skipLabelKey        = "eraser.sh/cleanup.skip"
		skipLabelValue      = "true"
	)

	pruneImagesFeat := features.New("Remove all non-running images from cluster").
		// Deploy 3 deployments with different images
		// We'll shutdown two of them, run eraser with `*`, then check that the images for the removed deployments are removed from the cluster.
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			nginxDep := util.NewDeployment(cfg.Namespace(), nginx, 2, map[string]string{"app": nginx}, corev1.Container{Image: nginx, Name: nginx})
			if err := cfg.Client().Resources().Create(ctx, nginxDep); err != nil {
				t.Error("Failed to create the nginx dep", err)
			}

			util.NewDeployment(cfg.Namespace(), redis, 2, map[string]string{"app": redis}, corev1.Container{Image: redis, Name: redis})
			err := cfg.Client().Resources().Create(ctx, util.NewDeployment(cfg.Namespace(), redis, 2, map[string]string{"app": redis}, corev1.Container{Image: redis, Name: redis}))
			if err != nil {
				t.Fatal(err)
			}

			util.NewDeployment(cfg.Namespace(), caddy, 2, map[string]string{"app": caddy}, corev1.Container{Image: caddy, Name: caddy})
			if err := cfg.Client().Resources().Create(ctx, util.NewDeployment(cfg.Namespace(), caddy, 2, map[string]string{"app": caddy}, corev1.Container{Image: caddy, Name: caddy})); err != nil {
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
				ObjectMeta: metav1.ObjectMeta{Name: nginx, Namespace: cfg.Namespace()},
			}

			if err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&nginxDep, appsv1.DeploymentAvailable, corev1.ConditionTrue),
				wait.WithTimeout(time.Minute*3)); err != nil {
				t.Fatal("nginx deployment not found", err)
			}
			ctx = context.WithValue(ctx, nginx, &nginxDep)

			redisDep := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: redis, Namespace: cfg.Namespace()},
			}
			if err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&redisDep, appsv1.DeploymentAvailable, corev1.ConditionTrue),
				wait.WithTimeout(time.Minute*3)); err != nil {
				t.Fatal("redis deployment not found", err)
			}
			ctx = context.WithValue(ctx, redis, &redisDep)

			caddyDep := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: caddy, Namespace: cfg.Namespace()},
			}
			if err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&caddyDep, appsv1.DeploymentAvailable, corev1.ConditionTrue),
				wait.WithTimeout(time.Minute*3)); err != nil {
				t.Fatal("caddy deployment not found", err)
			}
			ctx = context.WithValue(ctx, caddy, &caddyDep)

			return ctx
		}).
		Assess("Remove some of the deployments so the images aren't running", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Here we remove the redis and caddy deployments
			// Keep nginx running and ensure nginx is not deleted.
			var redisPods corev1.PodList
			if err := cfg.Client().Resources().List(ctx, &redisPods, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{"app": redis}).String()
			}); err != nil {
				t.Fatal(err)
			}
			if len(redisPods.Items) != 2 {
				t.Fatal("missing pods in redis deployment")
			}

			var caddyPods corev1.PodList
			if err := cfg.Client().Resources().List(ctx, &caddyPods, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{"app": caddy}).String()
			}); err != nil {
				t.Fatal(err)
			}
			if len(caddyPods.Items) != 2 {
				t.Fatal("missing pods in caddy deployment")
			}

			err := cfg.Client().Resources().Delete(ctx, ctx.Value(redis).(*appsv1.Deployment))
			if err != nil {
				t.Fatal(err)
			}
			err = cfg.Client().Resources().Delete(ctx, ctx.Value(caddy).(*appsv1.Deployment))
			if err != nil {
				t.Fatal(err)
			}

			for _, nodeName := range util.GetClusterNodes(t) {
				err := wait.For(util.ContainerNotPresentOnNode(nodeName, redis), wait.WithTimeout(time.Minute*2))
				if err != nil {
					// Let's not mark this as an error
					// We only have this to prevent race conditions with the eraser spinning up
					t.Logf("error while waiting for deployment deletion: %v", err)
				}
			}
			for _, nodeName := range util.GetClusterNodes(t) {
				err := wait.For(util.ContainerNotPresentOnNode(nodeName, caddy), wait.WithTimeout(time.Minute*2))
				if err != nil {
					// Let's not mark this as an error
					// We only have this to prevent race conditions with the eraser spinning up
					t.Logf("error while waiting for deployment deletion: %v", err)
				}
			}
			return ctx
		}).
		Assess("All non-running images are removed from the cluster", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			imgList := &eraserv1alpha1.ImageList{
				ObjectMeta: metav1.ObjectMeta{Name: prune},
				Spec: eraserv1alpha1.ImageListSpec{
					Images: []string{"*"},
				},
			}

			if err := cfg.Client().Resources().Create(ctx, imgList); err != nil {
				t.Fatal(err)
			}
			ctx = context.WithValue(ctx, prune, imgList)

			// The first check could take some extra time, where as things should be done already for the 2nd check.
			// So we'll give plenty of time and fail slow here.
			ctxT, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()
			util.CheckImageRemoved(ctxT, t, util.GetClusterNodes(t), redis)

			ctxT, cancel = context.WithTimeout(ctx, time.Minute)
			defer cancel()
			util.CheckImageRemoved(ctxT, t, util.GetClusterNodes(t), caddy)

			// Make sure nginx is still there
			util.CheckImagesExist(ctx, t, util.GetClusterNodes(t), nginx)

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if i := ctx.Value(nginx); i != nil {
				cfg.Client().Resources().Delete(ctx, i.(*appsv1.Deployment))
			}
			if i := ctx.Value(redis); i != nil {
				cfg.Client().Resources().Delete(ctx, i.(*appsv1.Deployment))
			}
			if i := ctx.Value(caddy); i != nil {
				cfg.Client().Resources().Delete(ctx, i.(*appsv1.Deployment))
			}
			if i := ctx.Value(prune); i != nil {
				cfg.Client().Resources().Delete(ctx, i.(*eraserv1alpha1.ImageList))
			}

			if err := util.DeleteImageListsAndJobs(cfg.KubeconfigFile()); err != nil {
				t.Error("Failed to clean eraser obejcts ", err)
			}

			// make sure nginx containers are cleaned up before proceeding
			for _, nodeName := range util.GetClusterNodes(t) {
				err := wait.For(util.ContainerNotPresentOnNode(nodeName, nginx), wait.WithTimeout(time.Minute*2))
				if err != nil {
					// Let's not mark this as an error
					// We only have this to prevent race conditions with the eraser spinning up
					t.Logf("error while waiting for deployment deletion: %v", err)
				}
			}

			return ctx
		}).
		Feature()

	testenv.Test(t, pruneImagesFeat)
}
