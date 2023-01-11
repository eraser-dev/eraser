package main

import (
	"context"

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
		CacheDir           string     `json:"cacheDir,omitempty"`
		DBRepo             string     `json:"dbRepo,omitempty"`
		DeleteFailedImages bool       `json:"deleteFailedImages,omitempty"`
		Vulnerabilities    VulnConfig `json:"vulnerabilities,omitempty"`
	}

	VulnConfig struct {
		IgnoreUnfixed  bool     `json:"ignoreUnfixed,omitempty"`
		Types          []string `json:"types,omitempty"`
		SecurityChecks []string `json:"securityChecks,omitempty"`
		Severities     []string `json:"severities,omitempty"`
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
	}
)

func DefaultConfig() *Config {
	return &Config{
		CacheDir:           "/var/lib/trivy",
		DBRepo:             "ghcr.io/aquasecurity/trivy.db",
		DeleteFailedImages: true,
		Vulnerabilities: VulnConfig{
			IgnoreUnfixed: true,
			Types: []string{
				vulnTypeOs,
				vulnTypeLibrary,
			},
			SecurityChecks: []string{securityCheckVuln},
			Severities:     []string{severityCritical},
		},
	}
}

type ImageScanner struct {
	trivyScanConfig    scannerSetup
	imageSourceOptions []fanalImage.Option
	userConfig         Config
}

var _ Scanner = &ImageScanner{}

// Function never returns an error.
func (s *ImageScanner) Scan(img unversioned.Image) (ScanStatus, error) {
	refs := make([]string, 0, len(img.Names)+len(img.Digests))
	refs = append(refs, img.Digests...)
	refs = append(refs, img.Names...)

	scanSucceeded := false
	log.Info("scanning image with id", "imageID", img.ImageID, "refs", refs)

	for i := 0; i < len(refs) && !scanSucceeded; i++ {
		ref := refs[i]
		log.Info("scanning image with ref", "ref", ref)

		dockerImage, cleanup, err := fanalImage.NewContainerImage(
			context.TODO(),
			ref,
			s.trivyScanConfig.dockerOptions,
			s.imageSourceOptions...,
		)
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
			RekorURL:          *rekorURL,
		})
		if err != nil {
			log.Error(err, "error registering config for artifact", "imageID", img.ImageID, "reference", ref)
			cleanup()
			continue
		}

		imageScanContext, cancel := context.WithTimeout(context.Background(), *imageScanTimeout)
		defer cancel()

		scanner := scanner.NewScanner(s.trivyScanConfig.localScanner, artifactToScan)
		report, err := scanner.ScanArtifact(imageScanContext, s.trivyScanConfig.scanOptions)
		if err != nil {
			log.Error(err, "error scanning image", "imageID", img.ImageID, "reference", ref)
			cleanup()
			continue
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
