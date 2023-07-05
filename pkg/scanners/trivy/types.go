package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
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

const (
	trivyJSONFormatFlag     = "--format=json"
	trivyImageArg           = "image"
	trivyCachDirFlag        = "--cache-dir"
	trivyTimeoutFlag        = "--timeout"
	trivyDBRepoFlag         = "--db-repository"
	trivyIgnoreUnfixedFlag  = "--ignore-unfixed"
	trivyVulTypesFlag       = "--vuln-type"
	trivySecurityChecksFlag = "--scanners"
	trivySeveritiesFlag     = "--severity"
	trivyRuntimeFlag        = "--image-src"
)

type (
	Config struct {
		Runtime            string        `json:"runtime,omitempty"`
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

func (c *Config) invocation(ref string) (string, []string) {
	args := []string{}

	// Global options
	args = append(args, trivyJSONFormatFlag)

	if c.CacheDir != "" {
		args = append(args, trivyCachDirFlag, c.CacheDir)
	}

	if c.Timeout.PerImage != 0 {
		args = append(args, trivyTimeoutFlag, time.Duration(c.Timeout.PerImage).String())
	}

	args = append(args, trivyImageArg, trivyRuntimeFlag, c.Runtime)

	// `trivy image`-specific options
	if c.DBRepo != "" {
		args = append(args, trivyDBRepoFlag, c.DBRepo)
	}

	if c.Vulnerabilities.IgnoreUnfixed {
		args = append(args, trivyIgnoreUnfixedFlag)
	}

	if len(c.Vulnerabilities.Types) > 0 {
		allVulnTypes := strings.Join(c.Vulnerabilities.Types, ",")
		args = append(args, trivyVulTypesFlag, allVulnTypes)
	}

	if len(c.Vulnerabilities.SecurityChecks) > 0 {
		allSecurityChecks := strings.Join(c.Vulnerabilities.SecurityChecks, ",")
		args = append(args, trivySecurityChecksFlag, allSecurityChecks)
	}

	if len(c.Vulnerabilities.Severities) > 0 {
		allSeverities := strings.Join(c.Vulnerabilities.Severities, ",")
		args = append(args, trivySeveritiesFlag, allSeverities)
	}

	args = append(args, ref)

	return "/trivy", args
}

type ImageScanner struct {
	trivyScanConfig    scannerSetup
	imageSourceOptions []fanalImage.Option
	userConfig         Config
	timer              *time.Timer
}

type ImageScanner2 struct {
	config Config
	timer  *time.Timer
}

func (s *ImageScanner2) Scan(img unversioned.Image) (ScanStatus, error) {
	refs := make([]string, 0, len(img.Names)+len(img.Digests))
	refs = append(refs, img.Digests...)
	refs = append(refs, img.Names...)

	perImageTimeout := time.Duration(s.config.Timeout.PerImage)
	_ = perImageTimeout
	scanSucceeded := false

	log.Info("scanning image with id", "imageID", img.ImageID, "refs", refs)

	for i := 0; i < len(refs) && !scanSucceeded; i++ {
		log.Info("scanning image with ref", "ref", refs[i])

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)

		cmdName, args := s.config.invocation(refs[i])
		cmd := exec.Command(cmdName, args...)
		cmd.Stdout = stdout
		cmd.Stderr = stderr

		// TODO: make this debug-only output
		log.Info("scanning image ref", "ref", refs[i], "cli_invocation", fmt.Sprintf("%s %s", cmdName, strings.Join(args, " ")))
		if err := cmd.Run(); err != nil {
			log.Error(err, "error scanning image", "imageID", img.ImageID, "reference", refs[i], "stderr", stderr.String())
			continue
		}

		var report trivyTypes.Report
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			log.Error(err, "error unmarshaling report", "imageID", img.ImageID, "reference", refs[i], "report", stdout.String(), "stderr", stderr.String())
			continue
		}

		if s.config.DeleteEOLImages {
			if report.Metadata.OS != nil && report.Metadata.OS.Eosl {
				log.Info("image is end of life", "imageID", img.ImageID, "reference", refs[i])
				return StatusNonCompliant, nil
			}
		}

		for j := range report.Results {
			if len(report.Results[j].Vulnerabilities) > 0 {
				// TODO: Surface vulnerability results
				return StatusNonCompliant, nil
			}
		}

		// causes a break from the loop
		scanSucceeded = true
	}

	status := StatusOK
	if !scanSucceeded {
		status = StatusFailed
	}

	return status, nil
}

func (s *ImageScanner2) Timer() *time.Timer {
	return s.timer
}

var _ Scanner = &ImageScanner{}

var _ Scanner = &ImageScanner2{}

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

		disabledAnalyzers := appendDisabledAnalyzers(analyzer.TypeConfigFiles, analyzer.TypeLockfiles, analyzer.TypeIndividualPkgs, analyzer.TypeLanguages)

		artifactToScan, err := artifactImage.NewArtifact(dockerImage, s.trivyScanConfig.fscache, artifact.Option{
			Offline:           true,
			DisabledAnalyzers: disabledAnalyzers,
			DisabledHandlers:  []fanalTypes.HandlerType{fanalTypes.UnpackagedPostHandler, fanalTypes.MisconfPostHandler},
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

func appendDisabledAnalyzers(analyzerType ...[]analyzer.Type) []analyzer.Type {
	var disableAnalyzers []analyzer.Type
	for _, v := range analyzerType {
		disableAnalyzers = append(disableAnalyzers, v...)
	}
	return disableAnalyzers
}
