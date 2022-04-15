//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	nginx = "nginx"
	redis = "redis"
	caddy = "caddy"

	prune = "imagelist"
)

func TestRemoveImagesFromAllNodes(t *testing.T) {

	rmImageFeat := features.New("Test Remove Image From All Nodes").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			podSelectorLabels := map[string]string{"app": nginx}
			nginxDep := newDeployment(cfg.Namespace(), nginx, 2, podSelectorLabels, corev1.Container{Image: nginx, Name: nginx})
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
				ObjectMeta: metav1.ObjectMeta{Name: nginx, Namespace: cfg.Namespace()},
			}

			if err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&resultDeployment, appsv1.DeploymentAvailable, corev1.ConditionTrue),
				wait.WithTimeout(time.Minute*1)); err != nil {
				t.Error("deployment not found", err)
			}

			return context.WithValue(ctx, nginx, &resultDeployment)
		}).
		Assess("Images successfully deleted from all nodes", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			//delete deployment
			client, err := cfg.NewClient()
			if err != nil {
				t.Error("Failed to create new client", err)
			}

			var pods corev1.PodList
			err = client.Resources().List(ctx, &pods, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(labels.Set{"app": nginx}).String()
			})
			if err != nil {
				t.Fatal(err)
			}

			dep := ctx.Value(nginx).(*appsv1.Deployment)
			if err := client.Resources().Delete(ctx, dep); err != nil {
				t.Error("Failed to delete the dep", err)
			}
			if err := wait.For(conditions.New(client.Resources()).ResourceDeleted(dep), wait.WithTimeout(time.Minute*1)); err != nil {
				// Let's not mark this as an error
				// We only have this to prevent race conditions with the eraser spinning up
				t.Logf("error while waiting for deployment deletion: %v", err)
			}
			if err := wait.For(conditions.New(client.Resources()).ResourcesDeleted(&pods), wait.WithTimeout(time.Minute)); err != nil {
				// Same as above, we aren't really interested in this error except for debugging problems later on.
				// We are only waiting for these pods so we don't hit race conditions with the eraser pod.
				t.Logf("error waiting for pods to be deleted: %v", err)
			}

			// deploy imageJob config
			if err := deployEraserConfig(cfg.KubeconfigFile(), "eraser-system", "test-data", "eraser_v1alpha1_imagelist.yaml"); err != nil {
				t.Error("Failed to deploy image list config", err)
			}

			ctxT, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()
			checkImageRemoved(ctxT, t, getClusterNodes(t), nginx)

			return ctx
		}).
		Assess("Pods from imagejobs are cleaned up", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Error("Failed to create new client", err)
			}

			var ls corev1.PodList
			err = c.Resources().List(ctx, &ls, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{"name": "eraser"}).String()
			})
			if err != nil {
				t.Errorf("could not list pods: %v", err)
			}

			err = wait.For(conditions.New(c.Resources()).ResourcesDeleted(&ls), wait.WithTimeout(time.Minute))
			if err != nil {
				t.Errorf("error waiting for pods to be deleted: %v", err)
			}

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := deleteEraserConfig(cfg.KubeconfigFile(), "eraser-system", "test-data", "eraser_v1alpha1_imagelist.yaml"); err != nil {
				t.Error("Failed to delete image list config ", err)
			}
			if err := KubectlDelete(cfg.KubeconfigFile(), "eraser-system", append([]string{"imagejob", "--all"})); err != nil {
				t.Error("Failed to delete image job(s) config ", err)
			}
			if err := KubectlDelete(cfg.KubeconfigFile(), "eraser-system", append([]string{"imagelist", "--all"})); err != nil {
				t.Error("Failed to delete image job(s) config ", err)
			}
			return ctx
		}).Feature()

	pruneImagesFeat := features.New("Remove all non-running images from cluster").
		// Deploy 3 deployments with different images
		// We'll shutdown two of them, run eraser with `*`, then check that the images for the removed deployments are removed from the cluster.
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			nginxDep := newDeployment(cfg.Namespace(), nginx, 2, map[string]string{"app": nginx}, corev1.Container{Image: nginx, Name: nginx})
			if err := cfg.Client().Resources().Create(ctx, nginxDep); err != nil {
				t.Error("Failed to create the nginx dep", err)
			}

			newDeployment(cfg.Namespace(), redis, 2, map[string]string{"app": redis}, corev1.Container{Image: redis, Name: redis})
			err := cfg.Client().Resources().Create(ctx, newDeployment(cfg.Namespace(), redis, 2, map[string]string{"app": redis}, corev1.Container{Image: redis, Name: redis}))
			if err != nil {
				t.Fatal(err)
			}

			newDeployment(cfg.Namespace(), caddy, 2, map[string]string{"app": caddy}, corev1.Container{Image: caddy, Name: caddy})
			if err := cfg.Client().Resources().Create(ctx, newDeployment(cfg.Namespace(), caddy, 2, map[string]string{"app": caddy}, corev1.Container{Image: caddy, Name: caddy})); err != nil {
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
				wait.WithTimeout(time.Minute*1)); err != nil {
				t.Fatal("nginx deployment not found", err)
			}
			ctx = context.WithValue(ctx, nginx, &nginxDep)

			redisDep := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: redis, Namespace: cfg.Namespace()},
			}
			if err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&redisDep, appsv1.DeploymentAvailable, corev1.ConditionTrue),
				wait.WithTimeout(time.Minute*1)); err != nil {
				t.Fatal("redis deployment not found", err)
			}
			ctx = context.WithValue(ctx, redis, &redisDep)

			caddyDep := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: caddy, Namespace: cfg.Namespace()},
			}
			if err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&caddyDep, appsv1.DeploymentAvailable, corev1.ConditionTrue),
				wait.WithTimeout(time.Minute*1)); err != nil {
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

			err = wait.For(conditions.New(cfg.Client().Resources()).ResourcesDeleted(&redisPods), wait.WithTimeout(time.Minute*1))
			if err != nil {
				t.Fatal(err)
			}
			err = wait.For(conditions.New(cfg.Client().Resources()).ResourcesDeleted(&caddyPods), wait.WithTimeout(time.Minute*1))
			if err != nil {
				t.Fatal(err)
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
			checkImageRemoved(ctxT, t, getClusterNodes(t), redis)

			ctxT, cancel = context.WithTimeout(ctx, time.Minute)
			defer cancel()
			checkImageRemoved(ctxT, t, getClusterNodes(t), caddy)

			// Make sure nginx is still there
			checkImagesExist(ctx, t, getClusterNodes(t), nginx)

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

			if err := KubectlDelete(cfg.KubeconfigFile(), "eraser-system", append([]string{"imagejob", "--all"})); err != nil {
				t.Error("Failed to delete image job(s) config ", err)
			}
			if err := KubectlDelete(cfg.KubeconfigFile(), "eraser-system", append([]string{"imagelist", "--all"})); err != nil {
				t.Error("Failed to delete image job(s) config ", err)
			}

			return ctx
		}).
		Feature()

	imglistChangeFeat := features.New("Test Updating ImageList to Reconcile").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Deploy 2 deployments with different images (nginx, redis)
			nginxDep := newDeployment(cfg.Namespace(), nginx, 2, map[string]string{"app": nginx}, corev1.Container{Image: nginx, Name: nginx})
			if err := cfg.Client().Resources().Create(ctx, nginxDep); err != nil {
				t.Error("Failed to create the nginx dep", err)
			}

			newDeployment(cfg.Namespace(), redis, 2, map[string]string{"app": redis}, corev1.Container{Image: redis, Name: redis})
			err := cfg.Client().Resources().Create(ctx, newDeployment(cfg.Namespace(), redis, 2, map[string]string{"app": redis}, corev1.Container{Image: redis, Name: redis}))
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
				ObjectMeta: metav1.ObjectMeta{Name: nginx, Namespace: cfg.Namespace()},
			}

			if err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&nginxDep, appsv1.DeploymentAvailable, corev1.ConditionTrue),
				wait.WithTimeout(time.Minute*1)); err != nil {
				t.Fatal("nginx deployment not found", err)
			}
			ctx = context.WithValue(ctx, nginx, &nginxDep)

			redisDep := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: redis, Namespace: cfg.Namespace()},
			}
			if err = wait.For(conditions.New(client.Resources()).DeploymentConditionMatch(&redisDep, appsv1.DeploymentAvailable, corev1.ConditionTrue),
				wait.WithTimeout(time.Minute*1)); err != nil {
				t.Fatal("redis deployment not found", err)
			}
			ctx = context.WithValue(ctx, redis, &redisDep)

			return ctx
		}).
		Assess("Remove deployments so the images aren't running", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			var redisPods corev1.PodList
			if err := cfg.Client().Resources().List(ctx, &redisPods, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{"app": redis}).String()
			}); err != nil {
				t.Fatal(err)
			}
			if len(redisPods.Items) != 2 {
				t.Fatal("missing pods in redis deployment")
			}

			var nginxPods corev1.PodList
			if err := cfg.Client().Resources().List(ctx, &nginxPods, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{"app": nginx}).String()
			}); err != nil {
				t.Fatal(err)
			}
			if len(nginxPods.Items) != 2 {
				t.Fatal("missing pods in nginx deployment")
			}

			err := cfg.Client().Resources().Delete(ctx, ctx.Value(redis).(*appsv1.Deployment))
			if err != nil {
				t.Fatal(err)
			}
			err = cfg.Client().Resources().Delete(ctx, ctx.Value(nginx).(*appsv1.Deployment))
			if err != nil {
				t.Fatal(err)
			}

			err = wait.For(conditions.New(cfg.Client().Resources()).ResourcesDeleted(&redisPods), wait.WithTimeout(time.Minute*1))
			if err != nil {
				t.Fatal(err)
			}
			err = wait.For(conditions.New(cfg.Client().Resources()).ResourcesDeleted(&nginxPods), wait.WithTimeout(time.Minute*1))
			if err != nil {
				t.Fatal(err)
			}
			return ctx
		}).
		Assess("Deploy imagelist to remove nginx", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// deploy imageJob config
			if err := deployEraserConfig(cfg.KubeconfigFile(), "eraser-system", "test-data", "eraser_v1alpha1_imagelist.yaml"); err != nil {
				t.Error("Failed to deploy image list config", err)
			}

			ctxT, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()
			checkImageRemoved(ctxT, t, getClusterNodes(t), nginx)

			return ctx
		}).
		Assess("Update imagelist to prune rest of images", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// deploy imageJob config
			if err := deployEraserConfig(cfg.KubeconfigFile(), "eraser-system", "test-data", "eraser_v1alpha1_imagelist_updated.yaml"); err != nil {
				t.Error("Failed to deploy image list config", err)
			}

			ctxT, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()
			checkImageRemoved(ctxT, t, getClusterNodes(t), redis)

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if i := ctx.Value(nginx); i != nil {
				cfg.Client().Resources().Delete(ctx, i.(*appsv1.Deployment))
			}
			if i := ctx.Value(redis); i != nil {
				cfg.Client().Resources().Delete(ctx, i.(*appsv1.Deployment))
			}
			if i := ctx.Value("imagelist"); i != nil {
				cfg.Client().Resources().Delete(ctx, i.(*eraserv1alpha1.ImageList))
			}

			if err := KubectlDelete(cfg.KubeconfigFile(), "eraser-system", append([]string{"imagejob", "--all"})); err != nil {
				t.Error("Failed to delete image job(s) config ", err)
			}
			if err := KubectlDelete(cfg.KubeconfigFile(), "eraser-system", append([]string{"imagelist", "--all"})); err != nil {
				t.Error("Failed to delete image job(s) config ", err)
			}

			return ctx
		}).Feature()

	testenv.Test(t, rmImageFeat)
	testenv.Test(t, pruneImagesFeat)
	testenv.Test(t, imglistChangeFeat)
}
