//go:build e2e
// +build e2e

package e2e

import (
	"os"
	"testing"

	eraserv1alpha2 "github.com/Azure/eraser/api/v1alpha2"
	"github.com/Azure/eraser/test/e2e/util"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
)

func TestMain(m *testing.M) {
	utilruntime.Must(eraserv1alpha2.AddToScheme(scheme.Scheme))

	util.Testenv = env.NewWithConfig(envconf.New())
	// Create KinD Cluster

	removerImage := util.ParsedImages.RemoverImage
	managerImage := util.ParsedImages.ManagerImage
	eraserImage := util.ParsedImages.EraserImage

	util.Testenv.Setup(
		envfuncs.CreateKindClusterWithConfig(util.KindClusterName, util.NodeVersion, "../../kind-config.yaml"),
		envfuncs.CreateNamespace(util.TestNamespace),
		util.LoadImageToCluster(util.KindClusterName, util.ManagerImage, util.ManagerTarballPath),
		util.LoadImageToCluster(util.KindClusterName, util.RemoverImage, util.RemoverTarballPath),
		util.HelmDeployLatestEraserRelease(util.TestNamespace,
			"--set", util.ScannerEnable.Set("false"),
			"--set", util.CollectorEnable.Set("false"),
			"--set", util.EraserImageRepo.Set(eraserImage.Repo),
			"--set", util.EraserImageTag.Set(eraserImage.Tag),
		),
		util.UpgradeEraserHelm(util.TestNamespace,
			"--set", util.CollectorEnable.Set("false"),
			"--set", util.ScannerEnable.Set("false"),
			"--set", util.RemoverImageRepo.Set(removerImage.Repo),
			"--set", util.RemoverImageTag.Set(removerImage.Tag),
			"--set", util.ManagerImageRepo.Set(managerImage.Repo),
			"--set", util.ManagerImageTag.Set(managerImage.Tag),
			"--set", util.CleanupOnSuccessDelay.Set("1m"),
		),
	).Finish(
		envfuncs.DestroyKindCluster(util.KindClusterName),
	)
	os.Exit(util.Testenv.Run(m))
}
