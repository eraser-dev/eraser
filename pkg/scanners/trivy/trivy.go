package main

import (
	"context"
	"errors"
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
	config        = flag.String("config", "", "path to the configuration file")
	enableProfile = flag.Bool("enable-pprof", false, "enable pprof profiling")
	profilePort   = flag.Int("pprof-port", 6060, "port for pprof profiling. defaulted to 6060 if unspecified")

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

	err := logger.Configure()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error setting up logger: %s", err)
		os.Exit(generalErr)
	}

	log.Info("config", "config", *config)

	userConfig := *DefaultConfig()
	if *config != "" {
		var err error
		userConfig, err = loadConfig(*config)
		if err != nil {
			log.Error(err, "unable to read config")
			os.Exit(generalErr)
		}
	}

	log.Info("userConfig", "userConfig", userConfig)
	log.Info("userConfig", "USERCONFIG", fmt.Sprintf("%#v\n", userConfig))

	// Initializes logger and parses CLI options into hashmap configs
	err = initGlobals(&userConfig.Vulnerabilities)
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
		template.WithDeleteScanFailedImages(userConfig.DeleteFailedImages),
	)

	allImages, err := provider.ReceiveImages()
	if err != nil {
		log.Error(err, "unable to read images from provider")
		os.Exit(generalErr)
	}

	s, err := initScanner(userConfig)
	if err != nil {
		log.Error(err, "error initializing scanner")
	}

	vulnerableImages, failedImages, err := scan(s, allImages)
	if err != nil {
		log.Error(err, "total image scan timed out")
	}

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
func initGlobals(cfg *VulnConfig) error {
	if cfg == nil {
		return fmt.Errorf("valid configuration required")
	}

	allSetsOfCommaSeparatedOptions := []optionSet{
		{
			input: cfg.Severities,
			m:     severityMap,
		},
		{
			input: cfg.Types,
			m:     vulnTypeMap,
		},
		{
			input: cfg.SecurityChecks,
			m:     securityCheckMap,
		},
	}

	for _, oSet := range allSetsOfCommaSeparatedOptions {
		fillMap(oSet.input, oSet.m)
	}

	return nil
}

func fillMap(sl []string, m map[string]bool) {
	for _, s := range sl {
		m[s] = true
	}
}

func runProfileServer() {
	server := &http.Server{
		Addr:              fmt.Sprintf("localhost:%d", *profilePort),
		ReadHeaderTimeout: 3 * time.Second,
	}
	err := server.ListenAndServe()
	log.Error(err, "pprof server failed")
}

func initScanner(userConfig Config) (Scanner, error) {
	cacheDir := userConfig.CacheDir
	err := downloadAndInitDB(userConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize trivy db. cacheDir: %s, error: %w", cacheDir, err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("error setting up trivy logger: %w", err)
	}

	sugar := logger.Sugar()
	trivylogger.Logger = sugar

	vulnTypeList := trueMapKeys(vulnTypeMap)
	securityCheckList := trueMapKeys(securityCheckMap)

	scanConfig, err := setupScanner(cacheDir, vulnTypeList, securityCheckList)
	if err != nil {
		return nil, err
	}

	runtime := os.Getenv(utils.EnvEraserContainerRuntime)
	imageSourceOptions, ok := runtimeFanalOptionsMap[runtime]
	if !ok {
		return nil, fmt.Errorf("unable to determine runtime from environment: %w", err)
	}

	totalTimeout := time.Duration(userConfig.Timeout.Total)
	timer := time.NewTimer(totalTimeout)

	var s Scanner = &ImageScanner{
		trivyScanConfig:    scanConfig,
		imageSourceOptions: imageSourceOptions,
		userConfig:         userConfig,
		timer:              timer,
	}
	return s, nil
}

func scan(s Scanner, allImages []unversioned.Image) ([]unversioned.Image, []unversioned.Image, error) {
	vulnerableImages := make([]unversioned.Image, 0, len(allImages))
	failedImages := make([]unversioned.Image, 0, len(allImages))
	// track total scan job time

	for idx, img := range allImages {
		select {
		case <-s.Timer().C:
			failedImages = append(failedImages, allImages[idx:]...)
			return vulnerableImages, failedImages, errors.New("image scan total timeout exceeded")
		default:
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
	}

	return vulnerableImages, failedImages, nil
}
