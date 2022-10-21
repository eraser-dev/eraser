//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/eraser/test/e2e/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"strconv"
	"strings"
)

func TestMetrics(t *testing.T) {
	metrics := features.New("ImagesRemoved and VulnerableImages metrics should report 1").
		Assess("Alpine image is removed", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// deploy imagelist config
			if err := util.DeployEraserConfig(cfg.KubeconfigFile(), util.TestNamespace, "../../test-data", "imagelist_alpine.yaml"); err != nil {
				t.Error("Failed to deploy image list config", err)
			}

			ctxT, cancel := context.WithTimeout(ctx, time.Minute*5)
			defer cancel()
			util.CheckImageRemoved(ctxT, t, util.GetClusterNodes(t), util.VulnerableImage)

			return ctx
		}).
		Assess("Check ImagesRemoved metric", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			c, err := cfg.NewClient()
			if err != nil {
				t.Fatal("Failed to create new client", err)
			}

			var ls corev1.PodList
			err = c.Resources().List(ctx, &ls, func(o *metav1.ListOptions) {
				o.LabelSelector = labels.SelectorFromSet(map[string]string{"component": "otel-collector"}).String()
			})
			if err != nil {
				t.Errorf("could not list pods: %v", err)
			}

			otelcollector := ls.Items[0]

			output, err := util.KubectlLogs(cfg.KubeconfigFile(), otelcollector.Name, "", util.TestNamespace)
			if err != nil {
				t.Errorf("could not get otelcollector logs: %v", err)
			}

			split := strings.Split(output, "}")

			count := 0
			for _, s := range split {
				if strings.Contains(s, "ImagesRemoved") {
					temp := strings.Split(s, "Value: ")[1]
					value := strings.Split(temp, "\\n")[0]

					v, err := strconv.Atoi(value)
					if err != nil {
						t.Error("could not covert metrics value to int")
					}
					count += v
				}
			}

			if count != 3 {
				t.Error("ImagesRemoved is not 3: ", count)
			}

			return ctx
		}).
		Feature()

	util.Testenv.Test(t, metrics)
}
