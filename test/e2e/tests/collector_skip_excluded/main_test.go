//go:build e2e
// +build e2e

package e2e

import (
	"os"
	"testing"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/test/e2e/util"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"

	pkgUtil "github.com/Azure/eraser/pkg/utils"
)

func TestMain(m *testing.M) {
	utilruntime.Must(eraserv1alpha1.AddToScheme(scheme.Scheme))

	eraserImage := util.ParsedImages.EraserImage
	managerImage := util.ParsedImages.ManagerImage
	collectorImage := util.ParsedImages.CollectorImage

	util.Testenv = env.NewWithConfig(envconf.New())
	// Create KinD Cluster
	util.Testenv.Setup(
		envfuncs.CreateKindClusterWithConfig(util.KindClusterName, util.NodeVersion, "../../kind-config.yaml"),
		envfuncs.CreateNamespace(util.TestNamespace),
		envfuncs.CreateNamespace(util.EraserNamespace),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, util.ManagerImage),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, util.Image),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, util.CollectorImage),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, util.VulnerableImage),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, util.NonVulnerableImage),
		util.CreateExclusionList(util.EraserNamespace, pkgUtil.ExclusionList{
			Excluded: []string{"docker.io/library/alpine:*"},
		}),
		util.CreateExclusionList(util.EraserNamespace, pkgUtil.ExclusionList{
			Excluded: []string{util.NonVulnerableImage},
		}),
		util.DeployEraserHelm(util.EraserNamespace,
			"--set", util.ScannerImageRepo.Set(""),
			"--set", util.EraserImageRepo.Set(eraserImage.Repo),
			"--set", util.EraserImageTag.Set(eraserImage.Tag),
			"--set", util.CollectorImageRepo.Set(collectorImage.Repo),
			"--set", util.CollectorImageTag.Set(collectorImage.Tag),
			"--set", util.ManagerImageRepo.Set(managerImage.Repo),
			"--set", util.ManagerImageTag.Set(managerImage.Tag),
			"--set", `controllerManager.additionalArgs={--job-cleanup-on-success-delay=1m}`),
	).Finish(
		envfuncs.DestroyKindCluster(util.KindClusterName),
	)
	os.Exit(util.Testenv.Run(m))
}
