package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	trivyTypes "github.com/aquasecurity/trivy/pkg/types"
	"github.com/eraser-dev/eraser/api/unversioned"
	"github.com/eraser-dev/eraser/pkg/utils"
)

const (
	StatusFailed ScanStatus = iota
	StatusNonCompliant
	StatusOK
	ImgSrcPodman     = "podman"
	ImgSrcDocker     = "docker"
	ImgSrcContainerd = "containerd"
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
	trivyIgnoreStatusFlag   = "--ignore-status"
)

type (
	Config struct {
		Runtime            unversioned.RuntimeSpec `json:"runtime,omitempty"`
		CacheDir           string                  `json:"cacheDir,omitempty"`
		DBRepo             string                  `json:"dbRepo,omitempty"`
		DeleteFailedImages bool                    `json:"deleteFailedImages,omitempty"`
		DeleteEOLImages    bool                    `json:"deleteEOLImages,omitempty"`
		Vulnerabilities    VulnConfig              `json:"vulnerabilities,omitempty"`
		Timeout            TimeoutConfig           `json:"timeout,omitempty"`
	}

	VulnConfig struct {
		IgnoreUnfixed   bool     `json:"ignoreUnfixed,omitempty"`
		Types           []string `json:"types,omitempty"`
		SecurityChecks  []string `json:"securityChecks,omitempty"`
		Severities      []string `json:"severities,omitempty"`
		IgnoredStatuses []string `json:"ignoredStatuses,omitempty"`
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
		Runtime: unversioned.RuntimeSpec{
			Name:    unversioned.RuntimeContainerd,
			Address: utils.CRIPath,
		},
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
			SecurityChecks:  []string{securityCheckVuln},
			Severities:      []string{severityCritical, severityHigh, severityMedium, severityLow},
			IgnoredStatuses: []string{},
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

	runtimeVar, err := c.getRuntimeVar()
	if err != nil {
		log.Error(err, "invalid runtime provided")
	}

	args = append(args, trivyImageArg, trivyRuntimeFlag, runtimeVar)

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

	if len(c.Vulnerabilities.IgnoredStatuses) > 0 {
		allIgnoredStatuses := strings.Join(c.Vulnerabilities.IgnoredStatuses, ",")
		args = append(args, trivyIgnoreStatusFlag, allIgnoredStatuses)
	}

	args = append(args, ref)

	return args
}

func (c *Config) getRuntimeVar() (string, error) {
	var imgsrc string
	runtimeName := c.Runtime.Name
	switch runtimeName {
	case unversioned.RuntimeCrio:
		imgsrc = ImgSrcPodman
	case unversioned.RuntimeDockerShim:
		imgsrc = ImgSrcDocker
	case unversioned.RuntimeContainerd, unversioned.Runtime(""):
		imgsrc = ImgSrcContainerd
	default:
		return "", fmt.Errorf("invalid runtime provided: %q", runtimeName)
	}
	return imgsrc, nil
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
		cmd.Env = append(cmd.Env, os.Environ()...)
		cmd.Env = setRuntimeSocketEnvVars(cmd, s.config.Runtime)

		log.V(1).Info("scanning image ref", "ref", refs[i], "cli_invocation", fmt.Sprintf("%s %s", trivyCommandName, strings.Join(cliArgs, " ")), "env", cmd.Env)
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

func setRuntimeSocketEnvVars(cmd *exec.Cmd, runtime unversioned.RuntimeSpec) []string {
	envKey := "CONTAINERD_ADDRESS"
	envVal := utils.CRIPath

	switch runtime.Name {
	case unversioned.RuntimeDockerShim:
		envKey = "DOCKER_HOST"
	case unversioned.RuntimeCrio:
		infoParent, err := os.Stat("/run/cri")
		if err != nil {
			log.Error(err, "unable to get permissions for cri directory")
		}

		infoSocket, err := os.Stat(utils.CRIPath)
		if err != nil {
			log.Error(err, "unable to get permissions for cri socket")
		}

		if err := os.Mkdir("/run/podman", infoParent.Mode().Perm()); err != nil {
			log.Error(err, "unable to create /run/podman dir")
		}

		if err := os.Symlink(utils.CRIPath, "/run/podman/podman.sock"); err != nil {
			log.Error(err, "unable to create symlink between CRI path and /run/podman/podman.sock")
		}

		if err := os.Chmod("/run/podman/podman.sock", infoSocket.Mode().Perm()); err != nil {
			log.Error(err, "unable to change /run/podman/podman.sock permissions")
		}
		envKey = "XDG_RUNTIME_DIR"
		envVal = "/run"
	}

	return append(cmd.Env, fmt.Sprintf("%s=%s", envKey, envVal))
}

func (s *ImageScanner) Timer() *time.Timer {
	return s.timer
}

var _ Scanner = &ImageScanner{}
