package config

import (
	"fmt"
	"sync"
	"time"

	v1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/version"
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

type Manager struct {
	mtx sync.Mutex
	cfg *v1alpha1.EraserConfig
}

func (m *Manager) Read() (v1alpha1.EraserConfig, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if m.cfg == nil {
		return v1alpha1.EraserConfig{}, fmt.Errorf("ConfigManager configuration is nil, aborting")
	}

	cfg := *m.cfg
	return cfg, nil
}

func (m *Manager) Update(newC *v1alpha1.EraserConfig) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if m.cfg == nil {
		return fmt.Errorf("ConfigManager configuration is nil, aborting")
	}

	if newC == nil {
		return fmt.Errorf("new configuration is nil, aborting")
	}

	*m.cfg = *newC
	return nil
}

func NewManager(cfg *v1alpha1.EraserConfig) *Manager {
	return &Manager{
		mtx: sync.Mutex{},
		cfg: cfg,
	}
}

const (
	noDelay = v1alpha1.Duration(0)
	oneDay  = v1alpha1.Duration(time.Hour * 24)
)

func Default() *v1alpha1.EraserConfig {
	return &v1alpha1.EraserConfig{
		Manager: v1alpha1.ManagerConfig{
			Runtime:      "containerd",
			OTLPEndpoint: "",
			LogLevel:     "info",
			Scheduling: v1alpha1.ScheduleConfig{
				RepeatInterval:   v1alpha1.Duration(oneDay),
				BeginImmediately: true,
			},
			Profile: v1alpha1.ProfileConfig{
				Enabled: false,
				Port:    6060,
			},
			ImageJob: v1alpha1.ImageJobConfig{
				SuccessRatio: 1.0,
				Cleanup: v1alpha1.ImageJobCleanupConfig{
					DelayOnSuccess: noDelay,
					DelayOnFailure: oneDay,
				},
			},
			PullSecrets: []string{},
			NodeFilter: v1alpha1.NodeFilterConfig{
				Type: "exclude",
				Selectors: []string{
					"eraser.sh/cleanup.filter",
				},
			},
		},
		Components: v1alpha1.Components{
			Collector: v1alpha1.OptionalContainerConfig{
				Enabled: false,
				ContainerConfig: v1alpha1.ContainerConfig{
					Image: v1alpha1.RepoTag{
						Repo: repo("collector"),
						Tag:  version.BuildVersion,
					},
					Request: v1alpha1.ResourceRequirements{
						Mem: resource.MustParse("25Mi"),
						CPU: resource.MustParse("7m"),
					},
					Limit: v1alpha1.ResourceRequirements{
						Mem: resource.MustParse("500Mi"),
						CPU: resource.Quantity{},
					},
					Config: nil,
				},
			},
			Scanner: v1alpha1.OptionalContainerConfig{
				Enabled: false,
				ContainerConfig: v1alpha1.ContainerConfig{
					Image: v1alpha1.RepoTag{
						Repo: repo("eraser-trivy-scanner"),
						Tag:  version.BuildVersion,
					},
					Request: v1alpha1.ResourceRequirements{
						Mem: resource.MustParse("500Mi"),
						CPU: resource.MustParse("1000m"),
					},
					Limit: v1alpha1.ResourceRequirements{
						Mem: resource.MustParse("2Gi"),
						CPU: resource.MustParse("1500m"),
					},
					Config: &defaultScannerConfig,
				},
			},
			Eraser: v1alpha1.ContainerConfig{
				Image: v1alpha1.RepoTag{
					Repo: repo("eraser"),
					Tag:  version.BuildVersion,
				},
				Request: v1alpha1.ResourceRequirements{
					Mem: resource.MustParse("25Mi"),
					CPU: resource.MustParse("7m"),
				},
				Limit: v1alpha1.ResourceRequirements{
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
