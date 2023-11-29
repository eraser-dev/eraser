package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/eraser-dev/eraser/api/unversioned"
	"go.uber.org/zap"

	_ "net/http/pprof"

	trivylogger "github.com/aquasecurity/trivy/pkg/log"
	"github.com/eraser-dev/eraser/pkg/logger"
	"github.com/eraser-dev/eraser/pkg/scanners/template"
	"github.com/eraser-dev/eraser/pkg/utils"
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

	statusUnknown            = "unknown"
	statusAffected           = "affected"
	statusFixed              = "fixed"
	statusUnderInvestigation = "under_investigation"
	statusWillNotFix         = "will_not_fix"
	statusFixDeferred        = "fix_deferred"
	statusEndOfLife          = "end_of_life"
)

var (
	config        = flag.String("config", "", "path to the configuration file")
	enableProfile = flag.Bool("enable-pprof", false, "enable pprof profiling")
	profilePort   = flag.Int("pprof-port", 6060, "port for pprof profiling. defaulted to 6060 if unspecified")

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

	log.Info("trivy version", "trivy version", trivyVersion)
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

	log.V(1).Info("userConfig",
		"json", userConfig,
		"struct", fmt.Sprintf("%#v\n", userConfig),
	)

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
		template.WithDeleteEOLImages(userConfig.DeleteEOLImages),
		template.WithDeletePinnedImages(userConfig.DeletePinnedImages),
	)

	allImages, err := provider.ReceiveImages()
	if err != nil {
		log.Error(err, "unable to read images from provider")
		os.Exit(generalErr)
	}
	// TODO - 2. We could filter the pinned images out here, to not affect the template.

	s, err := initScanner(&userConfig)
	if err != nil {
		log.Error(err, "error initializing scanner")
	}

	// TODO: 4 options to decide on how we'd want to {filter out/handle} `pinned` images.
	//   1. Filter inside the `ReceiveImages` function, part of the scanner template, so `allImages` is all non-pinned.
	//   2. Filter `allImages` AFTER `ReceiveImages`, so we don't affect the scanner template function.
	//   3. During the `scan` (or `Scan`) function, check if the image is pinned and continue.
	//		- This could be the most performant, where we don't add extra filtering, and just `continue` during image scans.
	//		- We could decide here if we still want to scan the pinned image, even when we don't want to delete it.
	//   4. Filter inside the `SendImages` function, part of the scanner template, so the images sent to the eraser are non-pinned.
	//		- Not sure how our template works, and if we'd want to filter there so other implementations don't have to.
	//   Adding filtering (aside from step 3 where we `continue`) would add an extra O(n) time complexity to go through all images and filter.
	vulnerableImages, failedImages, err := scan(s, allImages)
	if err != nil {
		log.Error(err, "total image scan timed out")
	}

	log.Info("Vulnerable", "Images", vulnerableImages, "Total count", len(vulnerableImages))

	if len(failedImages) > 0 {
		log.Info("Failed", "Images", failedImages)
	}

	err = provider.SendImages(vulnerableImages, failedImages)
	if err != nil {
		log.Error(err, "unable to write images")
	}

	log.Info("scanning complete, waiting for remover to finish...")
	err = provider.Finish()
	if err != nil {
		log.Error(err, "unable to complete scanning process")
	}

	log.Info("remover job completed, shutting down...")
}

func runProfileServer() {
	server := &http.Server{
		Addr:              fmt.Sprintf("localhost:%d", *profilePort),
		ReadHeaderTimeout: 3 * time.Second,
	}
	err := server.ListenAndServe()
	log.Error(err, "pprof server failed")
}

func initScanner(userConfig *Config) (Scanner, error) {
	if userConfig == nil {
		return nil, fmt.Errorf("invalid trivy scanner config")
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("error setting up trivy logger: %w", err)
	}

	sugar := logger.Sugar()
	trivylogger.Logger = sugar

	runtime := os.Getenv(utils.EnvEraserContainerRuntime)
	if runtime == "" {
		runtime = utils.RuntimeContainerd
	}

	userConfig.Runtime = runtime
	totalTimeout := time.Duration(userConfig.Timeout.Total)
	timer := time.NewTimer(totalTimeout)

	var s Scanner = &ImageScanner{
		config: *userConfig,
		timer:  timer,
	}
	return s, nil
}

func scan(s Scanner, allImages []unversioned.Image) ([]unversioned.Image, []unversioned.Image, error) {
	vulnerableImages := make([]unversioned.Image, 0, len(allImages))
	failedImages := make([]unversioned.Image, 0, len(allImages))
	// track total scan job time

	for idx, img := range allImages {
		// TODO - 3. we could filter out Pinned images here by `continue`-ing. we'll need to be sure nothing wonky happens with pinned images on timeout.
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
