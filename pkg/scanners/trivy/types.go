package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Azure/eraser/api/unversioned"
	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	fanalImage "github.com/aquasecurity/trivy/pkg/fanal/image"
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
		Total    eraserv1alpha1.Duration `json:"total,omitempty"`
		PerImage eraserv1alpha1.Duration `json:"perImage,omitempty"`
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
			Total:    eraserv1alpha1.Duration(time.Hour * 23),
			PerImage: eraserv1alpha1.Duration(time.Hour),
		},
	}
}

func (config *Config) Invocation() []string {
	args := []string{"trivy"}
	if config.CacheDir != "" {
		args = append(args, fmt.Sprintf("--cache-dir=%s", config.CacheDir))
	}

	var zero eraserv1alpha1.Duration
	if config.Timeout.PerImage != zero {
		args = append(args, fmt.Sprintf("--timeout=%s", time.Duration(config.Timeout.PerImage).String()))
	}

	args = append(args, "image", "--format=json")

	if config.DBRepo != "" {
		args = append(args, fmt.Sprintf("--db-repository=%s", config.DBRepo))
	}

	if config.Vulnerabilities.IgnoreUnfixed {
		args = append(args, "--ignore-unfixed")
	}

	if len(config.Vulnerabilities.Types) != 0 {
		s := strings.Join(config.Vulnerabilities.Types, ",")
		args = append(args, fmt.Sprintf("--vuln-type=%s", s))
	}

	if len(config.Vulnerabilities.SecurityChecks) != 0 {
		s := strings.Join(config.Vulnerabilities.SecurityChecks, ",")
		args = append(args, fmt.Sprintf("--scanners=%s", s))
	}

	if len(config.Vulnerabilities.Severities) != 0 {
		s := strings.Join(config.Vulnerabilities.Severities, ",")
		args = append(args, fmt.Sprintf("--severity=%s", s))
	}

	return args
}

type ImageScanner struct {
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

	// perImageTimeout := time.Duration(s.userConfig.Timeout.PerImage)

	scanSucceeded := false
	log.Info("scanning image with id", "imageID", img.ImageID, "refs", refs)

	for i := 0; i < len(refs) && !scanSucceeded; i++ {
		ref := refs[i]

		cmd := exec.Command("trivy", "image", "-f", "json", ref)
		stderr := new(bytes.Buffer)
		stdout := new(bytes.Buffer)
		cmd.Stderr = stderr
		cmd.Stdout = stdout

		if err := cmd.Run(); err != nil {
			log.Error(err, "could not scan image", "ref", ref, "stderr", stderr.String())
			continue
		}

		var report trivyTypes.Report
		if err := json.Unmarshal(stderr.Bytes(), &report); err != nil {
			log.Error(err, "unable to parse scan report", "report string", stdout.String())
			continue
		}

		if report.Metadata.OS != nil && report.Metadata.OS.Eosl {
			log.Info("image is end of life", "imageID", img.ImageID, "reference", ref)
			return StatusNonCompliant, nil
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

		// causes a break from the loop
		scanSucceeded = true
	}

	status := StatusOK
	if !scanSucceeded {
		status = StatusFailed
	}

	return status, nil
}
