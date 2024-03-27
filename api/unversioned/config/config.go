package config

import (
	"fmt"
	"sync"
	"time"

	"github.com/eraser-dev/eraser/api/unversioned"
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
ignoredStatuses:
`

type Manager struct {
	mtx sync.Mutex
	cfg *unversioned.EraserConfig
}

func (m *Manager) Read() (unversioned.EraserConfig, error) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	if m.cfg == nil {
		return unversioned.EraserConfig{}, fmt.Errorf("ConfigManager configuration is nil, aborting")
	}

	cfg := *m.cfg
	return cfg, nil
}

func (m *Manager) Update(newC *unversioned.EraserConfig) error {
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

func NewManager(cfg *unversioned.EraserConfig) *Manager {
	return &Manager{
		mtx: sync.Mutex{},
		cfg: cfg,
	}
}

const (
	noDelay = unversioned.Duration(0)
	oneDay  = unversioned.Duration(time.Hour * 24)
)

func Default() *unversioned.EraserConfig {
	return &unversioned.EraserConfig{
		Manager: unversioned.ManagerConfig{
			Runtime: unversioned.RuntimeSpec{
				Name:    unversioned.RuntimeContainerd,
				Address: "unix:///run/containerd/containerd.sock",
			},
			OTLPEndpoint: "",
			LogLevel:     "info",
			Scheduling: unversioned.ScheduleConfig{
				RepeatInterval:   unversioned.Duration(oneDay),
				BeginImmediately: true,
			},
			Profile: unversioned.ProfileConfig{
				Enabled: false,
				Port:    6060,
			},
			ImageJob: unversioned.ImageJobConfig{
				SuccessRatio: 1.0,
				Cleanup: unversioned.ImageJobCleanupConfig{
					DelayOnSuccess: noDelay,
					DelayOnFailure: oneDay,
				},
			},
			PullSecrets: []string{},
			NodeFilter: unversioned.NodeFilterConfig{
				Type: "exclude",
				Selectors: []string{
					"eraser.sh/cleanup.filter",
				},
			},
			AdditionalPodLabels: map[string]string{},
		},
		Components: unversioned.Components{
			Collector: unversioned.OptionalContainerConfig{
				Enabled: false,
				ContainerConfig: unversioned.ContainerConfig{
					Image: unversioned.RepoTag{
						Repo: repo("collector"),
						Tag:  version.BuildVersion,
					},
					Request: unversioned.ResourceRequirements{
						Mem: resource.MustParse("25Mi"),
						CPU: resource.MustParse("7m"),
					},
					Limit: unversioned.ResourceRequirements{
						Mem: resource.MustParse("500Mi"),
						CPU: resource.Quantity{},
					},
					Config: nil,
				},
			},
			Scanner: unversioned.OptionalContainerConfig{
				Enabled: false,
				ContainerConfig: unversioned.ContainerConfig{
					Image: unversioned.RepoTag{
						Repo: repo("eraser-trivy-scanner"),
						Tag:  version.BuildVersion,
					},
					Request: unversioned.ResourceRequirements{
						Mem: resource.MustParse("500Mi"),
						CPU: resource.MustParse("1000m"),
					},
					Limit: unversioned.ResourceRequirements{
						Mem: resource.MustParse("2Gi"),
						CPU: resource.MustParse("1500m"),
					},
					Config: &defaultScannerConfig,
				},
			},
			Remover: unversioned.ContainerConfig{
				Image: unversioned.RepoTag{
					Repo: repo("remover"),
					Tag:  version.BuildVersion,
				},
				Request: unversioned.ResourceRequirements{
					Mem: resource.MustParse("25Mi"),
					CPU: resource.MustParse("7m"),
				},
				Limit: unversioned.ResourceRequirements{
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
