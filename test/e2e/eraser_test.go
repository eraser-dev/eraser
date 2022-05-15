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
	clientgo "k8s.io/client-go/kubernetes"

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
				wait.WithTimeout(time.Minute*3)); err != nil {
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

			for _, nodeName := range getClusterNodes(t) {
				err := wait.For(containerNotPresentOnNode(nodeName, nginx), wait.WithTimeout(time.Minute*2))
				if err != nil {
					// Let's not mark this as an error
					// We only have this to prevent race conditions with the eraser spinning up
					t.Logf("error while waiting for deployment deletion: %v", err)
				}
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

			for _, nodeName := range getClusterNodes(t) {
				err := wait.For(containerNotPresentOnNode(nodeName, redis), wait.WithTimeout(time.Minute*2))
				if err != nil {
					// Let's not mark this as an error
					// We only have this to prevent race conditions with the eraser spinning up
					t.Logf("error while waiting for deployment deletion: %v", err)
				}
			}
			for _, nodeName := range getClusterNodes(t) {
				err := wait.For(containerNotPresentOnNode(nodeName, caddy), wait.WithTimeout(time.Minute*2))
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

			// make sure nginx containers are cleaned up before proceeding
			for _, nodeName := range getClusterNodes(t) {
				err := wait.For(containerNotPresentOnNode(nodeName, nginx), wait.WithTimeout(time.Minute*2))
				if err != nil {
					// Let's not mark this as an error
					// We only have this to prevent race conditions with the eraser spinning up
					t.Logf("error while waiting for deployment deletion: %v", err)
				}
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

			return ctx
		}).
		Assess("Remove deployments so the images aren't running", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Here we remove the redis and nginx deployments
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

			for _, nodeName := range getClusterNodes(t) {
				err := wait.For(containerNotPresentOnNode(nodeName, redis), wait.WithTimeout(time.Minute*2))
				if err != nil {
					// Let's not mark this as an error
					// We only have this to prevent race conditions with the eraser spinning up
					t.Logf("error while waiting for deployment deletion: %v", err)
				}
			}
			for _, nodeName := range getClusterNodes(t) {
				err := wait.For(containerNotPresentOnNode(nodeName, nginx), wait.WithTimeout(time.Minute*2))
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

	aliasFix := features.New("Specifying an image alias in the image list will delete the underlying image").
		// Deploy 3 deployments with different images
		// We'll shutdown two of them, run eraser with `*`, then check that the images for the removed deployments are removed from the cluster.
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Ensure that both nginx:one and nginx:two are tags for the same image digest
			_, err := dockerPullImage(nginxLatest)
			if err != nil {
				t.Error("failed to pull nginx image", err)
			}

			// Create the alias nginx:one
			_, err = dockerTagImage(nginxLatest, nginxAliasOne)
			if err != nil {
				t.Error("failed to tag nginx image", err)
			}

			// Create the alias nginx:two
			_, err = dockerTagImage(nginxLatest, nginxAliasTwo)
			if err != nil {
				t.Error("failed to tag nginx image", err)
			}

			// Load the images into the cluster
			_, err = kindLoadImage(kindClusterName, nginxAliasOne)
			if err != nil {
				t.Error("failed to load kind image", err)
			}

			_, err = kindLoadImage(kindClusterName, nginxAliasTwo)
			if err != nil {
				t.Error("failed to load kind image", err)
			}

			// Schedule two pods on a single node. Both pods will create containers from the same image,
			// but each pod refers to that same image by a different tag.
			nodeName := getClusterNodes(t)[0]
			nginxOnePod := newPod(cfg.Namespace(), nginxAliasOne, "nginxone", nodeName)
			ctx = context.WithValue(ctx, "nodeName", nodeName)

			if err := cfg.Client().Resources().Create(ctx, nginxOnePod); err != nil {
				t.Error("Failed to create the nginx pod", err)
			}
			ctx = context.WithValue(ctx, nginxAliasOne, nginxOnePod)

			nginxTwoPod := newPod(cfg.Namespace(), nginxAliasTwo, "nginxtwo", nodeName)
			if err := cfg.Client().Resources().Create(ctx, nginxTwoPod); err != nil {
				t.Error("Failed to create the nginx pod", err)
			}
			ctx = context.WithValue(ctx, nginxAliasTwo, nginxTwoPod)

			return ctx
		}).
		Assess("Pods successfully deployed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := cfg.NewClient()
			if err != nil {
				t.Error("Failed to create new client", err)
			}

			resultPod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "nginxone", Namespace: cfg.Namespace()},
			}

			err = wait.For(conditions.New(client.Resources()).PodConditionMatch(&resultPod, corev1.PodReady, corev1.ConditionTrue), wait.WithTimeout(time.Minute*3))
			if err != nil {
				t.Error("pod not deployed", err)
			}

			resultPod = corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "nginxtwo", Namespace: cfg.Namespace()},
			}

			err = wait.For(conditions.New(client.Resources()).PodConditionMatch(&resultPod, corev1.PodReady, corev1.ConditionTrue), wait.WithTimeout(time.Minute*3))
			if err != nil {
				t.Error("pod not deployed", err)
			}

			// Delete the pods, so they will be cleaned up
			nginxOnePod := ctx.Value(nginxAliasOne).(*corev1.Pod)
			if err := client.Resources().Delete(ctx, nginxOnePod); err != nil {
				t.Error("Failed to delete the dep", err)
			}

			nodeName := ctx.Value("nodeName").(string)
			err = wait.For(containerNotPresentOnNode(nodeName, "nginxone"), wait.WithTimeout(time.Minute*2))
			if err != nil {
				// Let's not mark this as an error
				// We only have this to prevent race conditions with the eraser spinning up
				t.Logf("error while waiting for deployment deletion: %v", err)
			}

			nginxTwoPod := ctx.Value(nginxAliasTwo).(*corev1.Pod)
			if err := client.Resources().Delete(ctx, nginxTwoPod); err != nil {
				t.Error("Failed to delete the dep", err)
			}
			err = wait.For(containerNotPresentOnNode(nodeName, "nginxtwo"), wait.WithTimeout(time.Minute*2))
			if err != nil {
				// Let's not mark this as an error
				// We only have this to prevent race conditions with the eraser spinning up
				t.Logf("error while waiting for deployment deletion: %v", err)
			}

			return ctx
		}).
		Assess("Image deleted when referencing by alias", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			imgList := &eraserv1alpha1.ImageList{
				ObjectMeta: metav1.ObjectMeta{Name: prune},
				Spec: eraserv1alpha1.ImageListSpec{
					Images: []string{nginxAliasTwo},
				},
			}
			if err := cfg.Client().Resources().Create(ctx, imgList); err != nil {
				t.Fatal(err)
			}

			nodeName := ctx.Value("nodeName").(string)
			ctxT, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()
			checkImageRemoved(ctxT, t, []string{nodeName}, nginx)

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := KubectlDelete(cfg.KubeconfigFile(), "eraser-system", append([]string{"imagejob", "--all"})); err != nil {
				t.Error("Failed to delete image job(s) config ", err)
			}
			if err := KubectlDelete(cfg.KubeconfigFile(), "eraser-system", append([]string{"imagelist", "--all"})); err != nil {
				t.Error("Failed to delete image job(s) config ", err)
			}

			// make sure nginx containers are cleaned up before proceeding
			for _, nodeName := range getClusterNodes(t) {
				err := wait.For(containerNotPresentOnNode(nodeName, nginx), wait.WithTimeout(time.Minute*2))
				if err != nil {
					// Let's not mark this as an error
					// We only have this to prevent race conditions with the eraser spinning up
					t.Logf("error while waiting for deployment deletion: %v", err)
				}
			}

			return ctx
		}).
		Feature()

	skipNodesFeat := features.New("Test node skipping by applying label").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// fetch node info
			c := cfg.Client().RESTConfig()
			k8sClient, err := clientgo.NewForConfig(c)
			if err != nil {
				t.Error("unable to obtain k8s client from config", err)
			}

			podSelectorLabels := map[string]string{"app": nginx}
			nginxDep := newDeployment(cfg.Namespace(), nginx, 2, podSelectorLabels, corev1.Container{Image: nginx, Name: nginx})
			if err := cfg.Client().Resources().Create(ctx, nginxDep); err != nil {
				t.Error("Failed to create the dep", err)
			}

			nodeList, err := k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{LabelSelector: skippedNodeSelector})
			if err != nil {
				t.Errorf("unable to list node %s\n%#v", skippedNodeSelector, err)
			}

			if len(nodeList.Items) != 1 {
				t.Errorf("List operation for selector %s resulted in the wrong number of nodes", skippedNodeSelector)
			}

			nodeToSkip := &nodeList.Items[0]
			nodeToSkip.ObjectMeta.Labels[skipLabelKey] = skipLabelValue

			nodeToSkip, err = k8sClient.CoreV1().Nodes().Update(ctx, nodeToSkip, metav1.UpdateOptions{})
			if err != nil {
				t.Errorf("unable to update node %#v with label {%s: %s}\nerror: %#v", nodeToSkip, skipLabelKey, skipLabelValue, err)
			}

			return ctx
		}).
		Assess("Deployment and labelling the node have succeeded", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c := cfg.Client().RESTConfig()
			k8sClient, err := clientgo.NewForConfig(c)
			if err != nil {
				t.Error("unable to obtain k8s client from config", err)
			}

			err = wait.For(func() (bool, error) {
				nodeList, err := k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{LabelSelector: skipLabelKey})
				if err != nil {
					return false, err
				}

				return len(nodeList.Items) == 1, nil
			}, wait.WithTimeout(time.Minute))
			if err != nil {
				t.Errorf("error while waiting for selector%s to be added to node\n%#v", skippedNodeSelector, err)
			}

			resultDeployment := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: nginx, Namespace: cfg.Namespace()},
			}

			if err = wait.For(
				conditions.New(cfg.Client().Resources()).DeploymentConditionMatch(&resultDeployment, appsv1.DeploymentAvailable, corev1.ConditionTrue),
				wait.WithTimeout(time.Minute*3),
			); err != nil {
				t.Error("deployment not found", err)
			}

			return context.WithValue(ctx, nginx, &resultDeployment)
		}).
		Assess("Node(s) successfully skipped", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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

			clusterNodes := getClusterNodes(t)
			clusterNodes = deleteStringFromSlice(clusterNodes, skippedNodeName)

			for _, nodeName := range clusterNodes {
				err := wait.For(containerNotPresentOnNode(nodeName, nginx), wait.WithTimeout(time.Minute*2))
				if err != nil {
					// Let's not mark this as an error
					// We only have this to prevent race conditions with the eraser spinning up
					t.Logf("error while waiting for deployment deletion: %v", err)
				}
			}

			// deploy imageJob config
			if err := deployEraserConfig(cfg.KubeconfigFile(), "eraser-system", "test-data", "eraser_v1alpha1_imagelist.yaml"); err != nil {
				t.Error("Failed to deploy image list config", err)
			}

			ctxT, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()

			// ensure images are removed from all nodes except the one we are skipping. remove the node we are skipping from the list of nodes.

			checkImageRemoved(ctxT, t, clusterNodes, nginx)

			// Wait for the imagejob to be completed by checking for its nonexistence in the cluster
			err = wait.For(imagejobNotInCluster(cfg.KubeconfigFile()), wait.WithTimeout(time.Minute*2))
			if err != nil {
				t.Logf("error while waiting for imagejob cleanup: %v", err)
			}

			// the imagejob has done its work, so now we can check the node to make sure it didn't remove the image
			checkImagesExist(ctx, t, []string{skippedNodeName}, nginx)

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := deleteEraserConfig(cfg.KubeconfigFile(), "eraser-system", "test-data", "eraser_v1alpha1_imagelist.yaml"); err != nil {
				t.Error("Failed to delete image list config ", err)
			}

			c := cfg.Client().RESTConfig()
			k8sClient, err := clientgo.NewForConfig(c)
			if err != nil {
				t.Error("unable to obtain k8s client from config", err)
			}

			nodeList, err := k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{LabelSelector: skippedNodeSelector})
			if err != nil {
				t.Errorf("unable to list node %s\n%#v", skippedNodeSelector, err)
			}

			if len(nodeList.Items) != 1 {
				t.Errorf("List operation for selector %s resulted in the wrong number of nodes", skippedNodeSelector)
			}

			skippedNode := &nodeList.Items[0]
			delete(skippedNode.ObjectMeta.Labels, skipLabelKey)

			skippedNode, err = k8sClient.CoreV1().Nodes().Update(ctx, skippedNode, metav1.UpdateOptions{})
			if err != nil {
				t.Errorf("unable to remove label %s from node %#v\nerror: %#v", skipLabelKey, skippedNode, err)
			}

			err = wait.For(func() (bool, error) {
				nodeList, err = k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{LabelSelector: skipLabelKey})
				if err != nil {
					return false, err
				}

				return len(nodeList.Items) == 0, nil
			}, wait.WithTimeout(time.Minute))
			if err != nil {
				t.Errorf("error while waiting for selector%s to be removed from node\n%#v", skippedNodeSelector, err)
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

	testenv.Test(t, rmImageFeat)
	testenv.Test(t, pruneImagesFeat)
	testenv.Test(t, imglistChangeFeat)
	testenv.Test(t, aliasFix)
	testenv.Test(t, skipNodesFeat)
}
