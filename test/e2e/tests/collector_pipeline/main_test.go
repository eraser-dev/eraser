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

	remover := util.ParsedImages.RemoverImage
	collector := util.ParsedImages.CollectorImage
	scanner := util.ParsedImages.ScannerImage
	manager := util.ParsedImages.ManagerImage

	util.Testenv = env.NewWithConfig(envconf.New())
	// Create KinD Cluster
	util.Testenv.Setup(
		envfuncs.CreateKindClusterWithConfig(util.KindClusterName, util.NodeVersion, "../../kind-config.yaml"),
		envfuncs.CreateNamespace(util.EraserNamespace),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, util.ManagerImage),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, util.RemoverImage),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, util.ScannerImage),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, util.CollectorImage),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, util.VulnerableImage),
		util.MakeDeploy(map[string]string{
			"REMOVER_REPO":       remover.Repo,
			"MANAGER_REPO":       manager.Repo,
			"TRIVY_SCANNER_REPO": scanner.Repo,
			"COLLECTOR_REPO":     collector.Repo,
			"REMOVER_TAG":        remover.Tag,
			"MANAGER_TAG":        manager.Tag,
			"TRIVY_SCANNER_TAG":  scanner.Tag,
			"COLLECTOR_TAG":      collector.Tag,
		}),
	).Finish(
		envfuncs.DestroyKindCluster(util.KindClusterName),
	)
	os.Exit(util.Testenv.Run(m))
}
