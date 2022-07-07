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
	clientgo "k8s.io/client-go/kubernetes"

	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestIncludeNodes(t *testing.T) {
	includeNodesFeat := features.New("Applying the eraser.sh/cleanup.filter label to a node should only schedule ImageJob pods on that node").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// fetch node info
			c := cfg.Client().RESTConfig()
			k8sClient, err := clientgo.NewForConfig(c)
			if err != nil {
				t.Error("unable to obtain k8s client from config", err)
			}

			podSelectorLabels := map[string]string{"app": util.Nginx}
			nginxDep := util.NewDeployment(cfg.Namespace(), util.Nginx, 2, podSelectorLabels, corev1.Container{Image: util.Nginx, Name: util.Nginx})
			if err := cfg.Client().Resources().Create(ctx, nginxDep); err != nil {
				t.Error("Failed to create the dep", err)
			}

			nodeList, err := k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{LabelSelector: util.FilterNodeSelector})
			if err != nil {
				t.Errorf("unable to list node %s\n%#v", util.FilterNodeSelector, err)
			}

			if len(nodeList.Items) != 1 {
				t.Errorf("List operation for selector %s resulted in the wrong number of nodes", util.FilterNodeSelector)
			}

			nodeInclude := &nodeList.Items[0]
			nodeInclude.ObjectMeta.Labels[util.FilterLabelKey] = util.FilterLabelValue

			nodeInclude, err = k8sClient.CoreV1().Nodes().Update(ctx, nodeInclude, metav1.UpdateOptions{})
			if err != nil {
				t.Errorf("unable to update node %#v with label {%s: %s}\nerror: %#v", nodeInclude, util.FilterLabelKey, util.FilterLabelValue, err)
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
				nodeList, err := k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{LabelSelector: util.FilterLabelKey})
				if err != nil {
					return false, err
				}

				return len(nodeList.Items) == 1, nil
			}, wait.WithTimeout(time.Minute))
			if err != nil {
				t.Errorf("error while waiting for selector%s to be added to node\n%#v", util.FilterNodeSelector, err)
			}

			resultDeployment := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: util.Nginx, Namespace: cfg.Namespace()},
			}

			if err = wait.For(
				conditions.New(cfg.Client().Resources()).DeploymentConditionMatch(&resultDeployment, appsv1.DeploymentAvailable, corev1.ConditionTrue),
				wait.WithTimeout(time.Minute*3),
			); err != nil {
				t.Error("deployment not found", err)
			}

			return context.WithValue(ctx, util.Nginx, &resultDeployment)
		}).
		Assess("Node(s) successfully included", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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

			err = wait.For(util.ContainerNotPresentOnNode(util.FilterNodeName, util.Nginx), wait.WithTimeout(time.Minute*2))
			if err != nil {
				// Let's not mark this as an error
				// We only have this to prevent race conditions with the eraser spinning up
				t.Logf("error while waiting for deployment deletion: %v", err)
			}

			// deploy imageJob config
			if err = util.DeployEraserConfig(cfg.KubeconfigFile(), "eraser-system", "../../test-data", "eraser_v1alpha1_imagelist.yaml"); err != nil {
				t.Error("Failed to deploy image list config", err)
			}

			ctxT, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()

			// ensure image is removed from filtered node.
			util.CheckImageRemoved(ctxT, t, []string{util.FilterNodeName}, util.Nginx)

			// Wait for the imagejob to be completed by checking for its nonexistence in the cluster
			err = wait.For(util.ImagejobNotInCluster(cfg.KubeconfigFile()), wait.WithTimeout(time.Minute*2))
			if err != nil {
				t.Logf("error while waiting for imagejob cleanup: %v", err)
			}

			clusterNodes := util.GetClusterNodes(t)
			clusterNodes = util.DeleteStringFromSlice(clusterNodes, util.FilterNodeName)

			// the imagejob has done its work, so now we can check the node to make sure it didn't remove the images from the remaining nodes
			util.CheckImagesExist(ctx, t, clusterNodes, util.Nginx)

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			if err := util.DeleteEraserConfig(cfg.KubeconfigFile(), "eraser-system", "../test-data", "eraser_v1alpha1_imagelist.yaml"); err != nil {
				t.Error("Failed to delete image list config ", err)
			}

			c := cfg.Client().RESTConfig()
			k8sClient, err := clientgo.NewForConfig(c)
			if err != nil {
				t.Error("unable to obtain k8s client from config", err)
			}

			nodeList, err := k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{LabelSelector: util.FilterNodeSelector})
			if err != nil {
				t.Errorf("unable to list node %s\n%#v", util.FilterNodeSelector, err)
			}

			if len(nodeList.Items) != 1 {
				t.Errorf("List operation for selector %s resulted in the wrong number of nodes", util.FilterNodeSelector)
			}

			filterNode := &nodeList.Items[0]
			delete(filterNode.ObjectMeta.Labels, util.FilterLabelKey)

			filterNode, err = k8sClient.CoreV1().Nodes().Update(ctx, filterNode, metav1.UpdateOptions{})
			if err != nil {
				t.Errorf("unable to remove label %s from node %#v\nerror: %#v", util.FilterLabelKey, filterNode, err)
			}

			err = wait.For(func() (bool, error) {
				nodeList, err = k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{LabelSelector: util.FilterLabelKey})
				if err != nil {
					return false, err
				}

				return len(nodeList.Items) == 0, nil
			}, wait.WithTimeout(time.Minute))
			if err != nil {
				t.Errorf("error while waiting for selector%s to be removed from node\n%#v", util.FilterNodeSelector, err)
			}

			if err := util.KubectlDelete(cfg.KubeconfigFile(), "eraser-system", append([]string{"imagejob", "--all"})); err != nil {
				t.Error("Failed to delete image job(s) config ", err)
			}
			if err := util.KubectlDelete(cfg.KubeconfigFile(), "eraser-system", append([]string{"imagelist", "--all"})); err != nil {
				t.Error("Failed to delete image job(s) config ", err)
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, includeNodesFeat)
}
