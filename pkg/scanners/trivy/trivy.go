package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"

	_ "net/http/pprof"

	"github.com/Azure/eraser/pkg/logger"
	util "github.com/Azure/eraser/pkg/utils"
	artifactImage "github.com/aquasecurity/trivy/pkg/fanal/artifact/image"
	fanalImage "github.com/aquasecurity/trivy/pkg/fanal/image"
	trivylogger "github.com/aquasecurity/trivy/pkg/log"
	"github.com/aquasecurity/trivy/pkg/scanner"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	generalErr = 1

	severityCritical = "CRITICAL"
	severityHigh     = "HIGH"
	severityMedium   = "MEDIUM"
	severityLow      = "LOW"
	severityUnknown  = "UNKNOWN"

	vulnTypeOs      = "os"
	vulnTypeLibrary = "library"

	securityCheckVuln   = "vuln"
	securityCheckConfig = "config"
	securityCheckSecret = "secret"
)

var (
	cacheDir               = flag.String("cache-dir", "/var/lib/trivy", "path to the cache dir")
	enableProfile          = flag.Bool("enable-pprof", false, "enable pprof profiling")
	ignoreUnfixed          = flag.Bool("ignore-unfixed", true, "report only fixed vulnerabilities")
	profilePort            = flag.Int("pprof-port", 6060, "port for pprof profiling. defaulted to 6060 if unspecified")
	securityChecks         = flag.String("security-checks", "vuln", "comma-separated list of what security issues to detect")
	severity               = flag.String("severity", "CRITICAL", "list of severity levels to report")
	vulnTypes              = flag.String("vuln-type", "os,library", "comma separated list of vulnerability types")
	deleteScanFailedImages = flag.Bool("delete-scan-failed-images", true, "whether or not to delete images for which scanning has failed")

	// Will be modified by parseCommaSeparatedOptions() to reflect the
	// `severity` CLI flag These are the only recognized severities and the
	// keys of this map should never be modified.
	severityMap = map[string]bool{
		severityCritical: false,
		severityHigh:     false,
		severityMedium:   false,
		severityLow:      false,
		severityUnknown:  false,
	}

	// Will be modified by parseCommaSeparatedOptions() to reflect the
	// `security-checks` CLI flag These are the only recognized security checks
	// and the keys of this map should never be modified.
	securityCheckMap = map[string]bool{
		securityCheckVuln:   false,
		securityCheckSecret: false,
		securityCheckConfig: false,
	}

	// Will be modified by parseCommaSeparatedOptions()  to reflect the
	// `vuln-type` CLI flag These are the only recognized vulnerability types
	// and the keys of this map should never be modified.
	vulnTypeMap = map[string]bool{
		vulnTypeOs:      false,
		vulnTypeLibrary: false,
	}

	log = logf.Log.WithName("scanner").WithValues("provider", "trivy")

	// This can be overwritten by the linker.
	trivyVersion = "dev"
)

func main() {
	flag.Parse()
	ctx := context.Background()

	var err error

	if err = logger.Configure(); err != nil {
		fmt.Fprintln(os.Stderr, "error setting up logger:", err)
		os.Exit(generalErr)
	}

	// creating new logger for JSON output with trivy scanner as trivy logs are not JSON encoded
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error setting up trivy logger:", err)
		os.Exit(generalErr)
	}

	sugar := logger.Sugar()
	trivylogger.Logger = sugar

	if err := unix.Mkfifo(util.EraseCompleteScanPath, util.PipeMode); err != nil {
		log.Error(err, "failed to create pipe", "pipeName", util.EraseCompleteScanPath)
		os.Exit(1)
	}

	err = os.Chmod(util.EraseCompleteScanPath, 0o666)
	if err != nil {
		log.Error(err, "unable to enable pipe for writing", "pipeName", util.EraseCompleteScanPath)
		os.Exit(1)
	}

	if *enableProfile {
		go func() {
			server := &http.Server{
				Addr:              fmt.Sprintf("localhost:%d", *profilePort),
				ReadHeaderTimeout: 3 * time.Second,
			}
			err := server.ListenAndServe()
			log.Error(err, "pprof server failed")
		}()
	}

	allSetsOfCommaSeparatedOptions := []optionSet{
		{
			input: *severity,
			m:     severityMap,
		},
		{
			input: *vulnTypes,
			m:     vulnTypeMap,
		},
		{
			input: *securityChecks,
			m:     securityCheckMap,
		},
	}

	for _, oSet := range allSetsOfCommaSeparatedOptions {
		// note: this function has side effects and will modify the map supplied as the first argument
		err := parseCommaSeparatedOptions(oSet.m, oSet.input)
		if err != nil {
			log.Error(err, "unable to parse options")
			os.Exit(generalErr)
		}
	}

	allImages, err := util.ReadCollectScanPipe(ctx)
	if err != nil {
		log.Error(err, "unable to read images from collect scan pipe")
		os.Exit(generalErr)
	}

	err = downloadAndInitDB(*cacheDir)
	if err != nil {
		log.Error(err, "unable to initialize trivy db", "cacheDir", *cacheDir)
		os.Exit(generalErr)
	}

	vulnTypeList := trueMapKeys(vulnTypeMap)
	securityCheckList := trueMapKeys(securityCheckMap)

	scanConfig, err := setupScanner(*cacheDir, vulnTypeList, securityCheckList)
	if err != nil {
		log.Error(err, "unable to set up scanner configuration", "cacheDir", *cacheDir, "vulnTypeList", vulnTypeList, "securityCheckList", securityCheckList)
		os.Exit(generalErr)
	}

	vulnerableImages := make([]eraserv1alpha1.Image, 0, len(allImages))
	failedImages := make([]eraserv1alpha1.Image, 0, len(allImages))

	for _, img := range allImages {
		refs := make([]string, 0, len(img.Names)+len(img.Digests))
		refs = append(refs, img.Digests...)
		refs = append(refs, img.Names...)

		if len(refs) == 0 {
			log.Info("found image with no name", "img", img)
			failedImages = append(failedImages, img)
			continue
		}

		imageScanFailed := true
		log.Info("scanning image with id", "imageID", img.ImageID, "refs", refs)

		for i := 0; i < len(refs) && imageScanFailed; i++ {
			ref := refs[i]
			log.Info("scanning image with ref", "ref", ref)

			dockerImage, cleanup, err := fanalImage.NewContainerImage(ctx, ref, scanConfig.dockerOptions,
				fanalImage.DisablePodman(),
				fanalImage.DisableDockerd(),
				fanalImage.DisableRemote(),
			)
			if err != nil { // could not locate image
				log.Error(err, "could not find image by reference", "imageID", img.ImageID, "reference", ref)
				cleanup()
				continue
			}
			log.Info("found image with id under reference", "imageID", img.ImageID, "ref", ref)

			artifactToScan, err := artifactImage.NewArtifact(dockerImage, scanConfig.fscache, defaulArtifactOptions)
			if err != nil {
				log.Error(err, "error registering config for artifact", "imageID", img.ImageID, "reference", ref)
				cleanup()
				continue
			}

			scanner := scanner.NewScanner(scanConfig.localScanner, artifactToScan)
			report, err := scanner.ScanArtifact(ctx, scanConfig.scanOptions)
			if err != nil {
				log.Error(err, "error scanning image", "imageID", img.ImageID, "reference", ref)
				cleanup()
				continue
			}

			imageScanFailed = false
		outer:
			for i := range report.Results {
				for j := range report.Results[i].Vulnerabilities {
					if *ignoreUnfixed && report.Results[i].Vulnerabilities[j].FixedVersion == "" {
						continue
					}

					if report.Results[i].Vulnerabilities[j].Severity == "" {
						report.Results[i].Vulnerabilities[j].Severity = severityUnknown
					}

					if severityMap[report.Results[i].Vulnerabilities[j].Severity] {
						vulnerableImages = append(vulnerableImages, img)
						break outer
					}
				}
			}

			cleanup()
		}

		if imageScanFailed {
			failedImages = append(failedImages, img)
		}
	}

	if len(failedImages) > 0 {
		log.Info("Failed", "Images", failedImages)
	}

	log.Info("Vulnerable", "Images", vulnerableImages)

	// if deleteScanFailedImages is true, we want to pass failed images as vulnerable to be deleted
	if *deleteScanFailedImages {
		vulnerableImages = append(vulnerableImages, failedImages...)
	}

	// write vulnerable images to scanErase pipe for eraser to read
	if err := util.WriteScanErasePipe(vulnerableImages); err != nil {
		log.Error(err, "unable to write non-compliant images to scan erase pipe")
		os.Exit(generalErr)
	}

	file, err := os.OpenFile(util.EraseCompleteScanPath, os.O_RDONLY, 0)
	if err != nil {
		log.Error(err, "failed to open pipe", "pipeName", util.EraseCompleteScanPath)
		os.Exit(1)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		log.Error(err, "failed to read pipe", "pipeName", util.EraseCompleteScanPath)
		os.Exit(1)
	}

	file.Close()

	if string(data) != util.EraseCompleteMessage {
		log.Info("garbage in pipe", "pipeName", util.EraseCompleteScanPath, "in_pipe", string(data))
		os.Exit(1)
	}
	log.Info("scanning complete, exiting")
}
