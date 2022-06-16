//go:build collector
// +build collector

package collector

import (
	"context"
	"testing"
	"time"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/test/e2e/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"os"
	"path/filepath"
)

func TestRemoveImagesFromAllNodes(t *testing.T) {
	const (
		alpine = "alpine"
	)

	collectScanErasePipelineFeat := features.New("Test Remove Image From All Nodes").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			wd, err := os.Getwd()
			if err != nil {
				t.Error("Could not get wd")
			}

			providerResourceAbsolutePath, err := filepath.Abs(filepath.Join(wd, "/../../../", providerResourceDirectory, "eraser"))
			if err != nil {
				t.Error("Could not get provider resource absolute pathy")
			}
			// start deployment
			if err := util.HelmInstall(cfg.KubeconfigFile(), "eraser-system", []string{providerResourceAbsolutePath}); err != nil {
				t.Error("Unable to helm install deployment")
			}

			return ctx
		}).
		Assess("ImageCollector CR is generated", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Fatal("Failed to create new client", err)
			}

			resource := eraserv1alpha1.ImageCollector{}
			wait.For(func() (bool, error) {
				err := c.Resources().Get(ctx, "imagecollector-shared", "default", &resource)
				if err != nil {
					return false, err
				}

				if resource.ObjectMeta.Name == "imagecollector-shared" {
					return true, nil
				}

				return false, nil
			}, wait.WithTimeout(time.Minute*3))

			return ctx
		}).
		Assess("ImageList CR is generated", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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
		Assess("Images successfully deleted from all nodes", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			ctxT, cancel := context.WithTimeout(ctx, 3*time.Minute)
			defer cancel()
			util.CheckImageRemoved(ctxT, t, util.GetClusterNodes(t), alpine)

			return ctx
		}).
		Assess("Pods from imagejobs are cleaned up", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Fatal("Failed to create new client", err)
			}

			var ls corev1.PodList
			err = c.Resources().List(ctx, &ls, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{"name": "collector"}).String()
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
			err := util.HelmUninstall(cfg.KubeconfigFile(), "eraser-system", []string{})
			if err != nil {
				t.Error("Unable to uninstall deployment for teardown", err)
			}

			return ctx
		}).
		Feature()

	disableScanFeat := features.New("Test Scanner Disabled Prune").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			wd, err := os.Getwd()
			if err != nil {
				t.Error("Could not get working directory", err)
			}

			providerResourceAbsolutePath, err := filepath.Abs(filepath.Join(wd, "/../../../", providerResourceDirectory, "eraser"))
			if err != nil {
				t.Error("Unable to get provider resource absolute path", err)
			}

			err = util.HelmInstall(cfg.KubeconfigFile(), "eraser-system", []string{providerResourceAbsolutePath, "--set", "scanner.image.repository="})
			if err != nil {
				t.Error("Unable to install deployment with scanner disabled", err)
			}

			return ctx
		}).
		Assess("ImageCollector CR is generated", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Error("Failed to create new client", err)
			}

			imagecollector := eraserv1alpha1.ImageCollector{}
			wait.For(func() (bool, error) {
				err := c.Resources().Get(ctx, "imagecollector-shared", "default", &imagecollector)
				if err != nil {
					t.Error("Could not get imagecollector-shared")
				}

				if imagecollector.ObjectMeta.Name == "imagecollector-shared" {
					return true, nil
				}

				return false, nil
			}, wait.WithTimeout(time.Minute*3))

			return ctx
		}).
		Assess("ImageList Spec Contains Same Images As ImageCollecotor Shared", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
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
			err = c.Resources().Get(ctx, "imagecollector-shared", "default", &imagecollectorShared)
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
				t.Error(img + " added")
			}

			// verify the images in both lists match
			for _, img := range imagecollectorShared.Spec.Images {
				// check by digest as we add to imagelist by digest when pruning without scanner
				if _, contains := imagelistSpec[img.Digest]; contains {
					t.Error("imagelist spec does not match imagecollector-shared: ", img.Digest)
				}
			}

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			err := util.HelmUninstall(cfg.KubeconfigFile(), "eraser-system", []string{})
			if err != nil {
				t.Error("Unable to uninstall deployment for teardown", err)
			}
			return ctx
		}).
		Feature()

	testenv.Test(t, disableScanFeat)
	testenv.Test(t, collectScanErasePipelineFeat)
}
