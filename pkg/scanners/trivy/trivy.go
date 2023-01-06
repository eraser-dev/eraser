package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Azure/eraser/api/unversioned"
	"go.uber.org/zap"

	_ "net/http/pprof"

	"github.com/Azure/eraser/pkg/logger"
	"github.com/Azure/eraser/pkg/scanners/template"
	"github.com/Azure/eraser/pkg/utils"
	fanalImage "github.com/aquasecurity/trivy/pkg/fanal/image"
	trivylogger "github.com/aquasecurity/trivy/pkg/log"
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
	vulnDBRepository       = flag.String("db-repository", "ghcr.io/aquasecurity/trivy-db", "vulnerability database repository")
	rekorURL               = flag.String("rekor-url", "https://rekor.sigstore.dev", "Rekor URL")
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

	runtimeFanalOptionsMap = map[string][]fanalImage.Option{
		utils.RuntimeDocker: {
			fanalImage.DisableRemote(),
			fanalImage.DisableContainerd(),
			fanalImage.DisablePodman(),
		},
		utils.RuntimeContainerd: {
			fanalImage.DisableRemote(),
			fanalImage.DisableDockerd(),
			fanalImage.DisablePodman(),
		},
		utils.RuntimeCrio: {
			fanalImage.DisableRemote(),
			fanalImage.DisableContainerd(),
			fanalImage.DisableDockerd(),
		},
	}

	log = logf.Log.WithName("scanner").WithValues("provider", "trivy")

	// This can be overwritten by the linker.
	trivyVersion = "dev"
)

func main() {
	flag.Parse()

	// Initializes logger and parses CLI options into hashmap configs
	err := initGlobals()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error initializing options: %v", err)
		os.Exit(generalErr)
	}

	if *enableProfile {
		go runProfileServer()
	}

	recordMetrics := false
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		recordMetrics = true
	}

	ctx := context.Background()
	provider := template.NewImageProvider(
		template.WithContext(ctx),
		template.WithLogger(log),
		template.WithMetrics(recordMetrics),
		template.WithDeleteScanFailedImages(*deleteScanFailedImages),
	)

	allImages, err := provider.ReceiveImages()
	if err != nil {
		log.Error(err, "unable to read images from provider")
		os.Exit(generalErr)
	}

	s, err := initScanner(ctx)
	if err != nil {
		log.Error(err, "error initializing scanner")
	}

	vulnerableImages, failedImages := scan(s, allImages)
	log.Info("Vulnerable", "Images", vulnerableImages)

	if len(failedImages) > 0 {
		log.Info("Failed", "Images", failedImages)
	}

	err = provider.SendImages(vulnerableImages, failedImages)
	if err != nil {
		log.Error(err, "unable to write images")
	}

	log.Info("scanning complete, waiting for eraser to finish...")
	err = provider.Finish()
	if err != nil {
		log.Error(err, "unable to complete scanning process")
	}

	log.Info("eraser job completed, shutting down...")
}

// Initializes logger and parses CLI options into hashmap configs.
func initGlobals() error {
	err := logger.Configure()
	if err != nil {
		return fmt.Errorf("error setting up logger: %w", err)
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
		err := parseCommaSeparatedOptions(oSet.m, oSet.input)
		if err != nil {
			return fmt.Errorf("unable to parse options %w", err)
		}
	}

	return nil
}

func runProfileServer() {
	server := &http.Server{
		Addr:              fmt.Sprintf("localhost:%d", *profilePort),
		ReadHeaderTimeout: 3 * time.Second,
	}
	err := server.ListenAndServe()
	log.Error(err, "pprof server failed")
}

func initScanner(ctx context.Context) (Scanner, error) {
	err := downloadAndInitDB(*cacheDir)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize trivy db. cacheDir: %s, error: %w", *cacheDir, err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("error setting up trivy logger: %w", err)
	}

	sugar := logger.Sugar()
	trivylogger.Logger = sugar

	vulnTypeList := trueMapKeys(vulnTypeMap)
	securityCheckList := trueMapKeys(securityCheckMap)

	scanConfig, err := setupScanner(*cacheDir, vulnTypeList, securityCheckList)
	if err != nil {
		return nil, err
	}

	runtime := os.Getenv(utils.EnvEraserContainerRuntime)
	imageSourceOptions, ok := runtimeFanalOptionsMap[runtime]
	if !ok {
		return nil, fmt.Errorf("unable to determine runtime from environment: %w", err)
	}

	var s Scanner = &ImageScanner{
		ctx:                ctx,
		scanConfig:         scanConfig,
		imageSourceOptions: imageSourceOptions,
	}
	return s, nil
}

func scan(s Scanner, allImages []unversioned.Image) ([]unversioned.Image, []unversioned.Image) {
	vulnerableImages := make([]unversioned.Image, 0, len(allImages))
	failedImages := make([]unversioned.Image, 0, len(allImages))

	for _, img := range allImages {
		// Logs scan failures
		status, err := s.Scan(img)
		if err != nil {
			failedImages = append(failedImages, img)
			log.Error(err, "scan failed")
			continue
		}

		switch status {
		case StatusNonCompliant:
			log.Info("vulnerable image found", "img", img)
			vulnerableImages = append(vulnerableImages, img)
		case StatusFailed:
			failedImages = append(failedImages, img)
		}
	}

	return vulnerableImages, failedImages
}
