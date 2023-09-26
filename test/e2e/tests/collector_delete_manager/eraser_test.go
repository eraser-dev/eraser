//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"

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
	deleteManagerFeat := features.New("Deleting manager pod while current ImageJob is running should delete ImageJob").
		Assess("Delete controller-manager pod", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Fatal("Failed to create new client", err)
			}

			var ls corev1.PodList
			err = c.Resources().List(ctx, &ls, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{util.ManagerLabelKey: util.ManagerLabelValue}).String()
			})
			if err != nil {
				t.Errorf("could not list manager pods: %v", err)
			}

			if len(ls.Items) != 1 {
				t.Error("incorrect number of manager pods: ", len(ls.Items))
			}

			if err := util.KubectlDelete(cfg.KubeconfigFile(), util.TestNamespace, []string{"pod", ls.Items[0].Name}); err != nil {
				t.Error("unable to delete eraser-controller-manager pod")
			}

			return ctx
		}).
		Assess("Check ImageJob is deleted", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Fatal("Failed to create new client", err)
			}

			var ls eraserv1alpha1.ImageJobList
			err = c.Resources().List(ctx, &ls)
			if err != nil {
				t.Errorf("could not list ImageJob: %v", err)
			}

			if len(ls.Items) != 1 {
				t.Error("incorrect number of ImageJobs: ", len(ls.Items))
			}

			err = wait.For(conditions.New(c.Resources()).ResourcesDeleted(&ls), wait.WithTimeout(util.Timeout))
			if err != nil {
				t.Errorf("error waiting for pods to be deleted: %v", err)
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, deleteManagerFeat)
}
