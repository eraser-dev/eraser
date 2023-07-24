package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Azure/eraser/api/unversioned"
	trivyTypes "github.com/aquasecurity/trivy/pkg/types"
)

const (
	StatusFailed ScanStatus = iota
	StatusNonCompliant
	StatusOK
)

const (
	trivyCommandName        = "/trivy"
	trivyImageArg           = "image"
	trivyJSONFormatFlag     = "--format=json"
	trivyCacheDirFlag       = "--cache-dir"
	trivyTimeoutFlag        = "--timeout"
	trivyDBRepoFlag         = "--db-repository"
	trivyIgnoreUnfixedFlag  = "--ignore-unfixed"
	trivyVulnTypesFlag      = "--vuln-type"
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

func (c *Config) cliArgs(ref string) []string {
	args := []string{}

	// Global options
	args = append(args, trivyJSONFormatFlag)

	if c.CacheDir != "" {
		args = append(args, trivyCacheDirFlag, c.CacheDir)
	}

	if c.Timeout.PerImage != 0 {
		args = append(args, trivyTimeoutFlag, time.Duration(c.Timeout.PerImage).String())
	}

	runtime := "containerd"
	// `trivy image`-specific options
	if c.Runtime != "" {
		runtime = c.Runtime
	}

	args = append(args, trivyImageArg, trivyRuntimeFlag, runtime)

	if c.DBRepo != "" {
		args = append(args, trivyDBRepoFlag, c.DBRepo)
	}

	if c.Vulnerabilities.IgnoreUnfixed {
		args = append(args, trivyIgnoreUnfixedFlag)
	}

	if len(c.Vulnerabilities.Types) > 0 {
		allVulnTypes := strings.Join(c.Vulnerabilities.Types, ",")
		args = append(args, trivyVulnTypesFlag, allVulnTypes)
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

	return args
}

type ImageScanner struct {
	config Config
	timer  *time.Timer
}

func (s *ImageScanner) Scan(img unversioned.Image) (ScanStatus, error) {
	refs := make([]string, 0, len(img.Names)+len(img.Digests))
	refs = append(refs, img.Digests...)
	refs = append(refs, img.Names...)
	scanSucceeded := false

	log.Info("scanning image with id", "imageID", img.ImageID, "refs", refs)
	for i := 0; i < len(refs) && !scanSucceeded; i++ {
		log.Info("scanning image with ref", "ref", refs[i])

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)

		cliArgs := s.config.cliArgs(refs[i])
		cmd := exec.Command(trivyCommandName, cliArgs...)
		cmd.Stdout = stdout
		cmd.Stderr = stderr

		log.V(1).Info("scanning image ref", "ref", refs[i], "cli_invocation", fmt.Sprintf("%s %s", trivyCommandName, strings.Join(cliArgs, " ")))
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

func (s *ImageScanner) Timer() *time.Timer {
	return s.timer
}

var _ Scanner = &ImageScanner{}
