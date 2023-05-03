package main

import (
	"context"
	"time"

	"github.com/Azure/eraser/api/unversioned"
	"github.com/aquasecurity/trivy/pkg/fanal/analyzer"
	"github.com/aquasecurity/trivy/pkg/fanal/artifact"
	artifactImage "github.com/aquasecurity/trivy/pkg/fanal/artifact/image"
	"github.com/aquasecurity/trivy/pkg/fanal/cache"
	fanalImage "github.com/aquasecurity/trivy/pkg/fanal/image"
	fanalTypes "github.com/aquasecurity/trivy/pkg/fanal/types"
	"github.com/aquasecurity/trivy/pkg/scanner"
	"github.com/aquasecurity/trivy/pkg/scanner/local"
	trivyTypes "github.com/aquasecurity/trivy/pkg/types"
)

const (
	StatusFailed ScanStatus = iota
	StatusNonCompliant
	StatusOK
)

type (
	Config struct {
		CacheDir           string        `json:"cacheDir,omitempty"`
		DBRepo             string        `json:"dbRepo,omitempty"`
		DeleteFailedImages bool          `json:"deleteFailedImages,omitempty"`
		DeleteEOLImages    bool          `json:"deleteEOLImages,omitempty"`
		Vulnerabilities    VulnConfig    `json:"vulnerabilities,omitempty"`
		Timeout            TimeoutConfig `json:"timeout,omitempty"`
	}

	VulnConfig struct {
		IgnoreUnfixed  bool     `json:"ignoreUnfixed,omitempty"`
		Types          []string `json:"types,omitempty"`
		SecurityChecks []string `json:"securityChecks,omitempty"`
		Severities     []string `json:"severities,omitempty"`
	}

	TimeoutConfig struct {
		Total    unversioned.Duration `json:"total,omitempty"`
		PerImage unversioned.Duration `json:"perImage,omitempty"`
	}

	scannerSetup struct {
		fscache       cache.FSCache
		localScanner  local.Scanner
		scanOptions   trivyTypes.ScanOptions
		dockerOptions fanalTypes.DockerOption
	}

	optionSet struct {
		input []string
		m     map[string]bool
	}

	ScanStatus int

	Scanner interface {
		Scan(unversioned.Image) (ScanStatus, error)
		Timer() *time.Timer
	}
)

func DefaultConfig() *Config {
	return &Config{
		CacheDir:           "/var/lib/trivy",
		DBRepo:             "ghcr.io/aquasecurity/trivy-db",
		DeleteFailedImages: true,
		DeleteEOLImages:    true,
		Vulnerabilities: VulnConfig{
			IgnoreUnfixed: true,
			Types: []string{
				vulnTypeOs,
				vulnTypeLibrary,
			},
			SecurityChecks: []string{securityCheckVuln},
			Severities:     []string{severityCritical, severityHigh, severityMedium, severityLow},
		},
		Timeout: TimeoutConfig{
			Total:    unversioned.Duration(time.Hour * 23),
			PerImage: unversioned.Duration(time.Hour),
		},
	}
}

type ImageScanner struct {
	trivyScanConfig    scannerSetup
	imageSourceOptions []fanalImage.Option
	userConfig         Config
	timer              *time.Timer
}

var _ Scanner = &ImageScanner{}

func (s *ImageScanner) Timer() *time.Timer {
	return s.timer
}

// Function never returns an error.
func (s *ImageScanner) Scan(img unversioned.Image) (ScanStatus, error) {
	refs := make([]string, 0, len(img.Names)+len(img.Digests))
	refs = append(refs, img.Digests...)
	refs = append(refs, img.Names...)

	perImageTimeout := time.Duration(s.userConfig.Timeout.PerImage)

	scanSucceeded := false
	log.Info("scanning image with id", "imageID", img.ImageID, "refs", refs)

	for i := 0; i < len(refs) && !scanSucceeded; i++ {
		ref := refs[i]
		log.Info("scanning image with ref", "ref", ref)

		dockerImage, cleanup, err := fanalImage.NewContainerImage(context.Background(), ref, s.trivyScanConfig.dockerOptions, s.imageSourceOptions...)
		if err != nil {
			log.Error(err, "could not find image by reference", "imageID", img.ImageID, "reference", ref)
			cleanup()
			continue
		}
		log.Info("found image with id under reference", "imageID", img.ImageID, "ref", ref)

		artifactToScan, err := artifactImage.NewArtifact(dockerImage, s.trivyScanConfig.fscache, artifact.Option{
			Offline:           true,
			DisabledAnalyzers: analyzer.TypeLockfiles,
			DisabledHandlers:  []fanalTypes.HandlerType{fanalTypes.UnpackagedPostHandler},
			SBOMSources:       []string{},
		})
		if err != nil {
			log.Error(err, "error registering config for artifact", "imageID", img.ImageID, "reference", ref)
			cleanup()
			continue
		}

		imageScanContext, cancel := context.WithTimeout(context.Background(), perImageTimeout)
		defer cancel()

		scanner := scanner.NewScanner(s.trivyScanConfig.localScanner, artifactToScan)
		report, err := scanner.ScanArtifact(imageScanContext, s.trivyScanConfig.scanOptions)
		if err != nil {
			log.Error(err, "error scanning image", "imageID", img.ImageID, "reference", ref)
			cleanup()
			continue
		}

		if s.userConfig.DeleteEOLImages {
			if report.Metadata.OS != nil && report.Metadata.OS.Eosl {
				log.Info("image is end of life", "imageID", img.ImageID, "reference", ref)
				return StatusNonCompliant, nil
			}
		}

		for i := range report.Results {
			for j := range report.Results[i].Vulnerabilities {
				if s.userConfig.Vulnerabilities.IgnoreUnfixed && report.Results[i].Vulnerabilities[j].FixedVersion == "" {
					continue
				}

				if report.Results[i].Vulnerabilities[j].Severity == "" {
					report.Results[i].Vulnerabilities[j].Severity = severityUnknown
				}

				if severityMap[report.Results[i].Vulnerabilities[j].Severity] {
					return StatusNonCompliant, nil
				}
			}
		}

		cleanup()

		// causes a break from the loop
		scanSucceeded = true
	}

	status := StatusOK
	if !scanSucceeded {
		status = StatusFailed
	}

	return status, nil
}
