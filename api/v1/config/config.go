package config

import (
	"time"

	v1 "github.com/Azure/eraser/api/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	version = "latest"

	defaultScannerConfig = `
cacheDir: /var/lib/trivy
dbRepo: ghcr.io/aquasecurity/trivy-db
deleteFailedImages: true
vulnerabilities:
  ignoreUnfixed: true
  types:
    - os
    - library
securityChecks: # need to be documented; determined by trivy, not us
  - vuln
severities:
  - CRITICAL
`
)

const (
	noDelay = v1.Duration(0)
	oneDay  = v1.Duration(time.Hour * 24)
)

func Default() *v1.EraserConfig {
	return &v1.EraserConfig{
		Manager: v1.ManagerConfig{
			Runtime:      "containerd",
			OTLPEndpoint: "",
			LogLevel:     "info",
			Scheduling: v1.ScheduleConfig{
				RepeatInterval:   v1.Duration(oneDay),
				BeginImmediately: true,
			},
			Profile: v1.ProfileConfig{
				Enabled: false,
				Port:    6060,
			},
			ImageJob: v1.ImageJobConfig{
				SuccessRatio: 1.0,
				Cleanup: v1.ImageJobCleanupConfig{
					DelayOnSuccess: noDelay,
					DelayOnFailure: oneDay,
				},
			},
			PullSecrets: []string{},
			NodeFilter: v1.NodeFilterConfig{
				Type: "exclude",
				Selectors: []string{
					"eraser.sh/cleanup.filter",
				},
			},
		},
		Components: v1.Components{
			Collector: v1.OptionalContainerConfig{
				Enabled: false,
				ContainerConfig: v1.ContainerConfig{
					Image: v1.RepoTag{
						Repo: "ghcr.io/azure/eraser/collector",
						Tag:  version,
					},
					Request: v1.ResourceRequirements{
						Mem: resource.MustParse("25Mi"),
						CPU: resource.MustParse("7m"),
					},
					Limit: v1.ResourceRequirements{
						Mem: resource.MustParse("500Mi"),
						CPU: resource.Quantity{},
					},
					Config: nil,
				},
			},
			Scanner: v1.OptionalContainerConfig{
				Enabled: false,
				ContainerConfig: v1.ContainerConfig{
					Image: v1.RepoTag{
						Repo: "ghcr.io/azure/eraser/trivy-scanner",
						Tag:  version,
					},
					Request: v1.ResourceRequirements{
						Mem: resource.MustParse("500Mi"),
						CPU: resource.MustParse("1000m"),
					},
					Limit: v1.ResourceRequirements{
						Mem: resource.MustParse("2Gi"),
						CPU: resource.MustParse("1500m"),
					},
					Config: &defaultScannerConfig,
				},
			},
			Eraser: v1.ContainerConfig{
				Image: v1.RepoTag{
					Repo: "ghcr.io/azure/eraser/eraser",
					Tag:  version,
				},
				Request: v1.ResourceRequirements{
					Mem: resource.MustParse("25Mi"),
					CPU: resource.MustParse("7m"),
				},
				Limit: v1.ResourceRequirements{
					Mem: resource.MustParse("30Mi"),
					CPU: resource.Quantity{},
				},
				Config: nil,
			},
		},
	}
}
