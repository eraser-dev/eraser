//go:build collector
// +build collector

package collector

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

var (
	providerResourceDirectory = "manifest_staging/charts"
	providerResource          = "eraser.yaml"
	eraserNamespace           = "eraser-system"
	testenv                   env.Environment
	image                     = os.Getenv("IMAGE")
	managerImage              = os.Getenv("MANAGER_IMAGE")
	collectorImage            = os.Getenv("COLLECTOR_IMAGE")
	scannerImage              = os.Getenv("SCANNER_IMAGE")
	vulnerableImage           = os.Getenv("VULNERABLE_IMAGE")
	nodeVersion               = os.Getenv("NODE_VERSION")
)

func TestMain(m *testing.M) {
	utilruntime.Must(eraserv1alpha1.AddToScheme(scheme.Scheme))

	testenv = env.NewWithConfig(envconf.New())
	// Create KinD Cluster
	namespace := envconf.RandomName("eraser-ns", 16)
	testenv.Setup(
		envfuncs.CreateKindClusterWithConfig(util.KindClusterName, nodeVersion, "../kind-config.yaml"),
		envfuncs.CreateNamespace(namespace),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, managerImage),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, image),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, scannerImage),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, collectorImage),
		envfuncs.LoadDockerImageToCluster(util.KindClusterName, vulnerableImage),
	).Finish(
		envfuncs.DeleteNamespace(namespace),
	)
	os.Exit(testenv.Run(m))
}
