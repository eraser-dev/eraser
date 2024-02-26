package config

import (
	"fmt"
	"time"

	"github.com/eraser-dev/eraser/api/v1alpha2"
	"github.com/eraser-dev/eraser/version"
	"k8s.io/apimachinery/pkg/api/resource"
)

var defaultScannerConfig = `
cacheDir: /var/lib/trivy
dbRepo: ghcr.io/aquasecurity/trivy-db
deleteFailedImages: true
deleteEOLImages: true
vulnerabilities:
  ignoreUnfixed: true
  types:
    - os
    - library
securityChecks: # need to be documented; determined by trivy, not us
  - vuln
severities:
  - CRITICAL
  - HIGH
  - MEDIUM
  - LOW
`

const (
	noDelay = v1alpha2.Duration(0)
	oneDay  = v1alpha2.Duration(time.Hour * 24)
)

func Default() *v1alpha2.EraserConfig {
	return &v1alpha2.EraserConfig{
		Manager: v1alpha2.ManagerConfig{
			Runtime:      "containerd",
			OTLPEndpoint: "",
			LogLevel:     "info",
			Scheduling: v1alpha2.ScheduleConfig{
				RepeatInterval:   v1alpha2.Duration(oneDay),
				BeginImmediately: true,
			},
			Profile: v1alpha2.ProfileConfig{
				Enabled: false,
				Port:    6060,
			},
			ImageJob: v1alpha2.ImageJobConfig{
				SuccessRatio: 1.0,
				Cleanup: v1alpha2.ImageJobCleanupConfig{
					DelayOnSuccess: noDelay,
					DelayOnFailure: oneDay,
				},
			},
			PullSecrets: []string{},
			NodeFilter: v1alpha2.NodeFilterConfig{
				Type: "exclude",
				Selectors: []string{
					"eraser.sh/cleanup.filter",
				},
			},
		},
		Components: v1alpha2.Components{
			Collector: v1alpha2.OptionalContainerConfig{
				Enabled: false,
				ContainerConfig: v1alpha2.ContainerConfig{
					Image: v1alpha2.RepoTag{
						Repo: repo("collector"),
						Tag:  version.BuildVersion,
					},
					Request: v1alpha2.ResourceRequirements{
						Mem: resource.MustParse("25Mi"),
						CPU: resource.MustParse("7m"),
					},
					Limit: v1alpha2.ResourceRequirements{
						Mem: resource.MustParse("500Mi"),
						CPU: resource.Quantity{},
					},
					Config: nil,
				},
			},
			Scanner: v1alpha2.OptionalContainerConfig{
				Enabled: false,
				ContainerConfig: v1alpha2.ContainerConfig{
					Image: v1alpha2.RepoTag{
						Repo: repo("eraser-trivy-scanner"),
						Tag:  version.BuildVersion,
					},
					Request: v1alpha2.ResourceRequirements{
						Mem: resource.MustParse("500Mi"),
						CPU: resource.MustParse("1000m"),
					},
					Limit: v1alpha2.ResourceRequirements{
						Mem: resource.MustParse("2Gi"),
						CPU: resource.MustParse("1500m"),
					},
					Config: &defaultScannerConfig,
				},
			},
			Remover: v1alpha2.ContainerConfig{
				Image: v1alpha2.RepoTag{
					Repo: repo("remover"),
					Tag:  version.BuildVersion,
				},
				Request: v1alpha2.ResourceRequirements{
					Mem: resource.MustParse("25Mi"),
					CPU: resource.MustParse("7m"),
				},
				Limit: v1alpha2.ResourceRequirements{
					Mem: resource.MustParse("30Mi"),
					CPU: resource.Quantity{},
				},
				Config: nil,
			},
		},
	}
}

func repo(basename string) string {
	if version.DefaultRepo == "" {
		return basename
	}

	return fmt.Sprintf("%s/%s", version.DefaultRepo, basename)
}
