//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"

	eraserv1 "github.com/eraser-dev/eraser/api/v1"
	"github.com/eraser-dev/eraser/test/e2e/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

type nodeString string

const (
	nginxOneName            = "nginxone"
	nginxTwoName            = "nginxtwo"
	nodeNameKey  nodeString = "nodeName"
)

func TestEnsureAliasedImageRemoved(t *testing.T) {
	aliasFix := features.New("Specifying an image alias in the image list will delete the underlying image").
		// Deploy 3 deployments with different images
		// We'll shutdown two of them, run eraser with `*`, then check that the images for the removed deployments are removed from the cluster.
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Ensure that both nginx:one and nginx:two are tags for the same image digest
			_, err := util.DockerPullImage(util.NginxLatest)
			if err != nil {
				t.Error("failed to pull nginx image", err)
			}

			// Schedule two pods on a single node. Both pods will create containers from the same image,
			// but each pod refers to that same image by a different tag.
			nodeName := util.GetClusterNodes(t)[0]

			// At ghcr.io/eraser-dev/eraser/e2e-test/nginx there is a repository
			// containing three tags. The three tags are `latest`, `one` and
			// `two`. They are all aliases for the same image; only the name
			// differs. These images are maintained there in order to avoid
			// sideloading images into the kind cluster, which has a known bug
			// associated with it. See https://github.com/containerd/containerd/issues/7698
			// for more information.
			nginxOnePod := util.NewPod(cfg.Namespace(), util.NginxAliasOne, nginxOneName, nodeName)
			ctx = context.WithValue(ctx, nodeNameKey, nodeName)

			if err := cfg.Client().Resources().Create(ctx, nginxOnePod); err != nil {
				t.Error("Failed to create the nginx pod", err)
			}

			nginxTwoPod := util.NewPod(cfg.Namespace(), util.NginxAliasTwo, nginxTwoName, nodeName)
			if err := cfg.Client().Resources().Create(ctx, nginxTwoPod); err != nil {
				t.Error("Failed to create the nginx pod", err)
			}

			return ctx
		}).
		Assess("Pods successfully deployed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := cfg.NewClient()
			if err != nil {
				t.Error("Failed to create new client", err)
			}

			resultPod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: nginxOneName, Namespace: cfg.Namespace()},
			}

			err = wait.For(conditions.New(client.Resources()).PodConditionMatch(&resultPod, corev1.PodReady, corev1.ConditionTrue), wait.WithTimeout(util.Timeout))
			if err != nil {
				t.Error("pod not deployed", err)
			}

			resultPod = corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: nginxTwoName, Namespace: cfg.Namespace()},
			}

			err = wait.For(conditions.New(client.Resources()).PodConditionMatch(&resultPod, corev1.PodReady, corev1.ConditionTrue), wait.WithTimeout(util.Timeout))
			if err != nil {
				t.Error("pod not deployed", err)
			}

			return ctx
		}).
		Assess("Pods successfully deleted", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			var (
				nginxOnePod corev1.Pod
				nginxTwoPod corev1.Pod
			)

			client, err := cfg.NewClient()
			if err != nil {
				t.Error("Failed to create new client", err)
			}

			if err := client.Resources().Get(ctx, nginxOneName, util.TestNamespace, &nginxOnePod); err != nil {
				t.Error("Failed to get the pod", err)
			}

			if err := client.Resources().Get(ctx, nginxTwoName, util.TestNamespace, &nginxTwoPod); err != nil {
				t.Error("Failed to get the pod", err)
			}

			// Delete the pods, so they will be cleaned up
			if err := client.Resources().Delete(ctx, &nginxOnePod); err != nil {
				t.Error("Failed to delete the pod", err)
			}

			if err := client.Resources().Delete(ctx, &nginxTwoPod); err != nil {
				t.Error("Failed to delete the pod", err)
			}

			toDelete := corev1.PodList{
				Items: []corev1.Pod{nginxOnePod, nginxTwoPod},
			}
			err = wait.For(conditions.New(client.Resources()).ResourcesDeleted(&toDelete))
			if err != nil {
				t.Error("failed to delete pods", err)
			}

			nodeName, ok := ctx.Value(nodeNameKey).(string)
			if !ok {
				t.Error("something is terribly wrong with the nodeName value")
			}

			if err := wait.For(util.ContainerNotPresentOnNode(nodeName, nginxOneName), wait.WithTimeout(util.Timeout)); err != nil {
				// Let's not mark this as an error
				// We only have this to prevent race conditions with the eraser spinning up
				t.Logf("error while waiting for deployment deletion: %v", err)
			}

			if err := wait.For(util.ContainerNotPresentOnNode(nodeName, nginxTwoName), wait.WithTimeout(util.Timeout)); err != nil {
				// Let's not mark this as an error
				// We only have this to prevent race conditions with the eraser spinning up
				t.Logf("error while waiting for deployment deletion: %v", err)
			}

			return ctx
		}).
		Assess("Image deleted when referencing by alias", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			imgList := &eraserv1.ImageList{
				ObjectMeta: metav1.ObjectMeta{Name: util.Prune},
				Spec: eraserv1.ImageListSpec{
					Images: []string{util.NginxAliasTwo},
				},
			}
			if err := cfg.Client().Resources().Create(ctx, imgList); err != nil {
				t.Fatal(err)
			}

			nodeName, ok := ctx.Value(nodeNameKey).(string)
			if !ok {
				t.Error("something is terribly wrong with the nodeName value")
			}

			ctxT, cancel := context.WithTimeout(ctx, util.Timeout)
			defer cancel()
			util.CheckImageRemoved(ctxT, t, []string{nodeName}, util.Nginx)

			return ctx
		}).
		Assess("Get logs", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := util.GetPodLogs(t); err != nil {
				t.Error("error getting eraser pod logs", err)
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, aliasFix)
}
