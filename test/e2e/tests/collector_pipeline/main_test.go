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
)

func TestMain(m *testing.M) {
	utilruntime.Must(eraserv1alpha1.AddToScheme(scheme.Scheme))

	version := os.Getenv("VERSION")
	eraserRepo := util.ParsedImages.EraserImage.Repo
	collectorRepo := util.ParsedImages.CollectorImage.Repo
	scannerRepo := util.ParsedImages.ScannerImage.Repo
	managerRepo := util.ParsedImages.ManagerImage.Repo

	util.Testenv = env.NewWithConfig(envconf.New())
	// Create KinD Cluster
	util.Testenv.Setup(
		envfuncs.CreateKindClusterWithConfig(util.KindClusterName, util.NodeVersion, "../../kind-config.yaml"),
		envfuncs.CreateNamespace(util.EraserNamespace),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, util.ManagerImage),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, util.Image),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, util.ScannerImage),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, util.CollectorImage),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, util.VulnerableImage),
		util.MakeDeploy(map[string]string{
			"ERASER_REPO":        eraserRepo,
			"MANAGER_REPO":       managerRepo,
			"TRIVY_SCANNER_REPO": scannerRepo,
			"COLLECTOR_IMAGE":    collectorRepo,
			"VERSION":            version,
		}),
	).Finish(
		envfuncs.DestroyKindCluster(util.KindClusterName),
	)
	os.Exit(util.Testenv.Run(m))
}
