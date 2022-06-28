//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/test/e2e/util"

	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestRemoveImagesFromAllNodes(t *testing.T) {
	const (
		alpine = "alpine"
	)

	disableScanFeat := features.New("Test Scanner Disabled Prune").
		Assess("ImageCollector CR is generated", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Error("Failed to create new client", err)
			}

			imagecollector := eraserv1alpha1.ImageCollector{}
			wait.For(func() (bool, error) {
				err := c.Resources().Get(ctx, util.ImageCollectorShared, "default", &imagecollector)
				if err != nil {
					t.Error("Could not get imagecollector-shared")
				}

				if imagecollector.ObjectMeta.Name == util.ImageCollectorShared {
					return true, nil
				}

				return false, nil
			}, wait.WithTimeout(time.Minute*3))

			return ctx
		}).
		Assess("ImageList Spec Contains Same Images As ImageCollector Shared", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Error("Failed to create new client", err)
			}

			// verify imagelist created
			imagelist := eraserv1alpha1.ImageList{}
			wait.For(func() (bool, error) {
				err := c.Resources().Get(ctx, "imagelist", "default", &imagelist)
				if util.IsNotFound(err) {
					return false, nil
				}

				if err != nil {
					return false, err
				}

				if imagelist.ObjectMeta.Name == "imagelist" {
					return true, nil
				}

				return false, nil
			}, wait.WithTimeout(time.Minute*3))

			imagecollectorShared := eraserv1alpha1.ImageCollector{}
			err = c.Resources().Get(ctx, util.ImageCollectorShared, "default", &imagecollectorShared)
			if err != nil {
				t.Error("Could not get imagecollector-shared")
			}

			// verify imagecollector-shared status fields are empty
			if imagecollectorShared.Status.Vulnerable != nil || imagecollectorShared.Status.Failed != nil {
				t.Error("Scan job has run, should be disabled")
			}

			imagelistSpec := make(map[string]struct{}, len(imagelist.Spec.Images))
			for _, img := range imagelist.Spec.Images {
				imagelistSpec[img] = struct{}{}
			}

			// verify the images in both lists match
			for _, img := range imagecollectorShared.Spec.Images {
				// check by digest as we add to imagelist by digest when pruning without scanner
				if _, contains := imagelistSpec[img.Digest]; !contains {
					t.Error("imagelist spec does not match imagecollector-shared: ", img.Digest)
				}
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, disableScanFeat)
}
