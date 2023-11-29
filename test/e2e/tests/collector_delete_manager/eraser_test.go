//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/eraser-dev/eraser/test/e2e/util"

	eraserv1alpha1 "github.com/eraser-dev/eraser/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestDeleteManager(t *testing.T) {
	deleteManagerFeat := features.New("Deleting manager pod while current ImageJob is running should delete ImageJob and restart").
		Assess("Wait for eraser pods running", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Fatal("Failed to create new client", err)
			}

			err = wait.For(
				util.NumPodsPresentForLabel(ctx, c, 3, util.ImageJobTypeLabelKey+"="+util.CollectorLabel),
				wait.WithTimeout(time.Minute*2),
				wait.WithInterval(time.Millisecond*500),
			)
			if err != nil {
				t.Fatal(err)
			}

			return ctx
		}).
		Assess("Delete controller-manager pod", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Fatal("Failed to create new client", err)
			}

			// get manager pod
			var podList corev1.PodList
			err = c.Resources().List(ctx, &podList, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{util.ManagerLabelKey: util.ManagerLabelValue}).String()
			})
			if err != nil {
				t.Errorf("could not list manager pods: %v", err)
			}

			if len(podList.Items) != 1 {
				t.Error("incorrect number of manager pods: ", len(podList.Items))
			}

			// get current ImageJob before deleting manager pod
			var jobList eraserv1alpha1.ImageJobList
			err = c.Resources().List(ctx, &jobList)
			if err != nil {
				t.Errorf("could not list ImageJob: %v", err)
			}

			t.Log("job", jobList.Items[0], "name", jobList.Items[0].Name)

			if len(jobList.Items) != 1 {
				t.Error("incorrect number of ImageJobs: ", len(jobList.Items))
			}

			// delete manager pod
			if err := util.KubectlDelete(cfg.KubeconfigFile(), util.TestNamespace, []string{"pod", podList.Items[0].Name}); err != nil {
				t.Error("unable to delete eraser-controller-manager pod")
			}

			// wait for deletion of ImageJob
			err = wait.For(conditions.New(c.Resources()).ResourcesDeleted(&jobList), wait.WithTimeout(util.Timeout))
			if err != nil {
				t.Errorf("error waiting for ImageJob to be deleted: %v", err)
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, deleteManagerFeat)
}
