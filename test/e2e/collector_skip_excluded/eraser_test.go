//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/test/e2e/util"

	"strings"

	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestCollectorExcluded(t *testing.T) {
	collectorExcluded := features.New("ImageCollector should remove excluded images from imagecollector-shared").
		Assess("ImageCollector CR is generated", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Fatal("Failed to create new client", err)
			}

			resource := eraserv1alpha1.ImageCollector{}
			wait.For(func() (bool, error) {
				err := c.Resources().Get(ctx, util.ImageCollectorShared, "default", &resource)
				if err != nil {
					return false, err
				}

				if resource.ObjectMeta.Name == util.ImageCollectorShared {
					return true, nil
				}

				return false, nil
			}, wait.WithTimeout(time.Minute*3))

			return ctx
		}).
		Assess("ImageList CR is generated", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// check that imagelist CR is generated to make sure collection portion is completed
			c, err := cfg.NewClient()
			if err != nil {
				t.Fatal("Failed to create new client", err)
			}

			resource := eraserv1alpha1.ImageList{}
			wait.For(func() (bool, error) {
				err := c.Resources().Get(ctx, "imagelist", "default", &resource)
				if util.IsNotFound(err) {
					return false, nil
				}

				if err != nil {
					return false, err
				}

				if resource.ObjectMeta.Name == "imagelist" {
					return true, nil
				}

				return false, nil
			}, wait.WithTimeout(time.Minute*3))

			return ctx
		}).
		Assess("ImageCollector CR shared does not contain Alpine", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Fatal("Failed to create new client", err)
			}

			resource := eraserv1alpha1.ImageCollector{}
			wait.For(func() (bool, error) {
				err := c.Resources().Get(ctx, util.ImageCollectorShared, "default", &resource)
				if err != nil {
					return false, err
				}

				if resource.ObjectMeta.Name == util.ImageCollectorShared {
					return true, nil
				}

				return false, nil
			}, wait.WithTimeout(time.Minute*3))

			// alpine is excluded and should not be added to imagecollector-shared
			for _, img := range resource.Spec.Images {
				if strings.Contains(img.Name, "alpine") {
					t.Error("imagecollector-shared should not contain alpine", img.Name)
				}
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, collectorExcluded)
}
