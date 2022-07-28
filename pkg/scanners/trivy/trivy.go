package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	_ "net/http/pprof"

	"github.com/Azure/eraser/pkg/logger"
	"github.com/aquasecurity/fanal/artifact"
	artifactImage "github.com/aquasecurity/fanal/artifact/image"
	fanalImage "github.com/aquasecurity/fanal/image"
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

	apiPath         = "apis/eraser.sh/v1alpha1"
	resourceName    = "imagecollectors"
	subResourceName = "status"
)

var (
	cacheDir        = flag.String("cache-dir", "/var/lib/trivy", "path to the cache dir")
	collectorCRName = flag.String("collector-cr-name", "collector-cr", "name of the collector cr to read from and write to")
	enableProfile   = flag.Bool("enable-pprof", false, "enable pprof profiling")
	ignoreUnfixed   = flag.Bool("ignore-unfixed", true, "report only fixed vulnerabilities")
	profilePort     = flag.Int("pprof-port", 6060, "port for pprof profiling. defaulted to 6060 if unspecified")
	securityChecks  = flag.String("security-checks", "vuln", "comma-separated list of what security issues to detect")
	severity        = flag.String("severity", "CRITICAL", "list of severity levels to report")
	vulnTypes       = flag.String("vuln-type", "os,library", "comma separated list of vulnerability types")

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

	if err := logger.Configure(); err != nil {
		fmt.Fprintln(os.Stderr, "Error setting up logger:", err)
		os.Exit(generalErr)
	}

	if *enableProfile {
		go func() {
			err := http.ListenAndServe(fmt.Sprintf("localhost:%d", *profilePort), nil)
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

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Error(err, "unable to get in-cluster config")
		os.Exit(generalErr)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Error(err, "unable to get REST client")
		os.Exit(generalErr)
	}

	result := eraserv1alpha1.ImageCollector{}

	err = clientset.RESTClient().Get().
		AbsPath(apiPath).
		Resource(resourceName).
		Name(*collectorCRName).
		Do(context.Background()).
		Into(&result)
	if err != nil {
		log.Error(err, "RESTClient GET request failed", "apiPath", apiPath, "recourceName", resourceName, "collectorCRName", *collectorCRName)
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

	resultClient := initializeResultClient()
	vulnerableImages := make([]eraserv1alpha1.Image, 0, len(result.Spec.Images))
	failedImages := make([]eraserv1alpha1.Image, 0, len(result.Spec.Images))

	for k := range result.Spec.Images {
		img := result.Spec.Images[k]

		imageRef := img.Name
		if imageRef == "" {
			log.Info("found image with no name", "img", img)
			failedImages = append(failedImages, img)
			continue
		}

		log.Info("scanning image", "imageRef", imageRef)

		dockerImage, cleanup, err := fanalImage.NewDockerImage(ctx, imageRef, scanConfig.dockerOptions)
		if err != nil {
			log.Error(err, "error fetching manifest for image", "img", img)
			failedImages = append(failedImages, img)
			cleanup()
			continue
		}

		artifactToScan, err := artifactImage.NewArtifact(dockerImage, scanConfig.fscache, artifact.Option{})
		if err != nil {
			log.Error(err, "error registering config for artifact", "img", img)
			failedImages = append(failedImages, img)
			cleanup()
			continue
		}

		scanner := scanner.NewScanner(scanConfig.localScanner, artifactToScan)
		report, err := scanner.ScanArtifact(ctx, scanConfig.scanOptions)
		if err != nil {
			log.Error(err, "error scanning image", "img", img)
			failedImages = append(failedImages, img)
			cleanup()
			continue
		}

	outer:
		for i := range report.Results {
			resultClient.FillVulnerabilityInfo(report.Results[i].Vulnerabilities, report.Results[i].Type)

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

	err = updateStatus(&statusUpdate{
		apiPath:          apiPath,
		ctx:              ctx,
		clientset:        clientset,
		collectorCRName:  *collectorCRName,
		resourceName:     resourceName,
		subResourceName:  subResourceName,
		vulnerableImages: vulnerableImages,
		failedImages:     failedImages,
	})
	if err != nil {
		log.Error(err, "error updating ImageCollectorStatus", "images", vulnerableImages)
		os.Exit(generalErr)
	}

	log.Info("scanning complete, exiting")
}
