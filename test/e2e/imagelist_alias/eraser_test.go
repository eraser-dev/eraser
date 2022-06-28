//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/test/e2e/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

	aliasFix := features.New("Specifying an image alias in the image list will delete the underlying image").
		// Deploy 3 deployments with different images
		// We'll shutdown two of them, run eraser with `*`, then check that the images for the removed deployments are removed from the cluster.
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Ensure that both nginx:one and nginx:two are tags for the same image digest
			_, err := util.DockerPullImage(nginxLatest)
			if err != nil {
				t.Error("failed to pull nginx image", err)
			}

			// Create the alias nginx:one
			_, err = util.DockerTagImage(nginxLatest, nginxAliasOne)
			if err != nil {
				t.Error("failed to tag nginx image", err)
			}

			// Create the alias nginx:two
			_, err = util.DockerTagImage(nginxLatest, nginxAliasTwo)
			if err != nil {
				t.Error("failed to tag nginx image", err)
			}

			// Load the images into the cluster
			_, err = util.KindLoadImage(util.KindClusterName, nginxAliasOne)
			if err != nil {
				t.Error("failed to load kind image", err)
			}

			_, err = util.KindLoadImage(util.KindClusterName, nginxAliasTwo)
			if err != nil {
				t.Error("failed to load kind image", err)
			}

			// Schedule two pods on a single node. Both pods will create containers from the same image,
			// but each pod refers to that same image by a different tag.
			nodeName := util.GetClusterNodes(t)[0]
			nginxOnePod := util.NewPod(cfg.Namespace(), nginxAliasOne, "nginxone", nodeName)
			ctx = context.WithValue(ctx, "nodeName", nodeName)

			if err := cfg.Client().Resources().Create(ctx, nginxOnePod); err != nil {
				t.Error("Failed to create the nginx pod", err)
			}
			ctx = context.WithValue(ctx, nginxAliasOne, nginxOnePod)

			nginxTwoPod := util.NewPod(cfg.Namespace(), nginxAliasTwo, "nginxtwo", nodeName)
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
			err = wait.For(util.ContainerNotPresentOnNode(nodeName, "nginxone"), wait.WithTimeout(time.Minute*2))
			if err != nil {
				// Let's not mark this as an error
				// We only have this to prevent race conditions with the eraser spinning up
				t.Logf("error while waiting for deployment deletion: %v", err)
			}

			nginxTwoPod := ctx.Value(nginxAliasTwo).(*corev1.Pod)
			if err := client.Resources().Delete(ctx, nginxTwoPod); err != nil {
				t.Error("Failed to delete the dep", err)
			}
			err = wait.For(util.ContainerNotPresentOnNode(nodeName, "nginxtwo"), wait.WithTimeout(time.Minute*2))
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
			util.CheckImageRemoved(ctxT, t, []string{nodeName}, nginx)

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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

	testenv.Test(t, aliasFix)
}
