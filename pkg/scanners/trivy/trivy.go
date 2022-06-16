package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	machinerytypes "k8s.io/apimachinery/pkg/types"

	eraserv1alpha1 "github.com/Azure/eraser/api/eraser.sh/v1alpha1"
	clientset "github.com/Azure/eraser/pkg/client/clientset/versioned"

	"k8s.io/client-go/rest"

	_ "net/http/pprof"

	"github.com/Azure/eraser/pkg/logger"
	"github.com/aquasecurity/fanal/applier"
	"github.com/aquasecurity/fanal/artifact"
	artifactImage "github.com/aquasecurity/fanal/artifact/image"
	"github.com/aquasecurity/fanal/cache"
	fanalImage "github.com/aquasecurity/fanal/image"
	fanalTypes "github.com/aquasecurity/fanal/types"
	"github.com/aquasecurity/trivy-db/pkg/db"
	dlDb "github.com/aquasecurity/trivy/pkg/db"
	"github.com/aquasecurity/trivy/pkg/detector/ospkg"
	pkgResult "github.com/aquasecurity/trivy/pkg/result"
	"github.com/aquasecurity/trivy/pkg/scanner"
	"github.com/aquasecurity/trivy/pkg/scanner/local"
	trivyTypes "github.com/aquasecurity/trivy/pkg/types"
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
	collectorCRName = flag.String("collector-cr-name", "collector-cr", "name of the collector cr to read from and write to")
	cacheDir        = flag.String("cache-dir", "/var/lib/trivy", "path to the cache dir")
	severity        = flag.String("severity", "CRITICAL", "list of severity levels to report")
	ignoreUnfixed   = flag.Bool("ignore-unfixed", true, "report only fixed vulnerabilities")
	vulnTypes       = flag.String("vuln-type", "os,library", "comma separated list of vulnerability types")
	securityChecks  = flag.String("security-checks", "vuln,secret", "comma-separated list of what security issues to detect")
	enableProfile   = flag.Bool("enable-pprof", false, "enable pprof profiling")
	profilePort     = flag.Int("pprof-port", 6060, "port for pprof profiling. defaulted to 6060 if unspecified")

	// Will be modified by parseCommaSeparatedOptions() to reflect the `severity` CLI flag
	// These are the only recognized severities and the keys of this map should never be modified.
	severityMap = map[string]bool{
		severityCritical: false,
		severityHigh:     false,
		severityMedium:   false,
		severityLow:      false,
		severityUnknown:  false,
	}

	// Will be modified by parseCommaSeparatedOptions() to reflect the `security-checks` CLI flag
	// These are the only recognized security checks and the keys of this map should never be modified.
	securityCheckMap = map[string]bool{
		securityCheckVuln:   false,
		securityCheckSecret: false,
		securityCheckConfig: false,
	}

	// Will be modified by parseCommaSeparatedOptions()  to reflect the `vuln-type` CLI flag
	// These are the only recognized vulnerability types and the keys of this map should never be modified.
	vulnTypeMap = map[string]bool{
		vulnTypeOs:      false,
		vulnTypeLibrary: false,
	}

	log = logf.Log.WithName("scanner").WithValues("provider", "trivy")

	trivyVersion = "dev"
)

type (
	scannerSetup struct {
		fscache       cache.FSCache
		localScanner  local.Scanner
		scanOptions   trivyTypes.ScanOptions
		dockerOptions fanalTypes.DockerOption
	}

	patch struct {
		Status eraserv1alpha1.ImageCollectorStatus `json:"status"`
	}

	optionSet struct {
		input string
		m     map[string]bool
	}

	statusUpdate struct {
		apiPath          string
		ctx              context.Context
		clientset        *clientset.Clientset
		collectorCRName  string
		resourceName     string
		subResourceName  string
		vulnerableImages []eraserv1alpha1.Image
		failedImages     []eraserv1alpha1.Image
	}
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

	allCommaSeparatedOptions := []optionSet{
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

	for _, oSet := range allCommaSeparatedOptions {
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

	clientset, err := clientset.NewForConfig(cfg)
	if err != nil {
		log.Error(err, "unable to get REST client")
		os.Exit(generalErr)
	}

	result, err := clientset.EraserV1alpha1().ImageCollectors("default").Get(ctx, *collectorCRName, v1.GetOptions{})
	if err != nil {
		log.Error(err, "Typed client get failed", "apiPath", apiPath, "recourceName", resourceName, "collectorCRName", *collectorCRName, "collector", result)
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

// side effects: map `m` will be modified according to the values in `commaSeparatedList`.
func parseCommaSeparatedOptions(m map[string]bool, commaSeparatedList string) error {
	list := strings.Split(commaSeparatedList, ",")
	for _, item := range list {
		if _, ok := m[item]; !ok {
			keys := mapKeys(m)
			return fmt.Errorf("'%s' was not one of %#v", item, keys)
		}

		m[item] = true
	}

	return nil
}

func downloadAndInitDB(cacheDir string) error {
	err := downloadDB(cacheDir)
	if err != nil {
		return err
	}

	err = db.Init(cacheDir)
	if err != nil {
		return err
	}

	return nil
}

func downloadDB(cacheDir string) error {
	client := dlDb.NewClient(cacheDir, true)
	ctx := context.Background()
	needsUpdate, err := client.NeedsUpdate(trivyVersion, false)
	if err != nil {
		return err
	}

	if needsUpdate {
		if err = client.Download(ctx, cacheDir); err != nil {
			return err
		}
	}

	return nil
}

func setupScanner(cacheDir string, vulnTypes, securityChecks []string) (scannerSetup, error) {
	filesystemCache, err := cache.NewFSCache(cacheDir)
	if err != nil {
		return scannerSetup{}, err
	}

	app := applier.NewApplier(filesystemCache)
	det := ospkg.Detector{}
	dopts := fanalTypes.DockerOption{}
	scan := local.NewScanner(app, det)

	sopts := trivyTypes.ScanOptions{
		VulnType:            vulnTypes,
		SecurityChecks:      securityChecks,
		ScanRemovedPackages: false,
		ListAllPackages:     false,
	}

	return scannerSetup{
		localScanner:  scan,
		scanOptions:   sopts,
		dockerOptions: dopts,
		fscache:       filesystemCache,
	}, nil
}

func initializeResultClient() pkgResult.Client {
	config := db.Config{}
	client := pkgResult.NewClient(config)
	return client
}

func updateStatus(opts *statusUpdate) error {
	collectorPatch := patch{
		Status: eraserv1alpha1.ImageCollectorStatus{
			Vulnerable: opts.vulnerableImages,
			Failed:     opts.failedImages,
		},
	}

	body, err := json.Marshal(&collectorPatch)
	if err != nil {
		return err
	}

	_, err = opts.clientset.RESTClient().Patch(machinerytypes.MergePatchType).
		AbsPath(opts.apiPath).
		Resource(opts.resourceName).
		SubResource(opts.subResourceName).
		Name(opts.collectorCRName).
		Body(body).DoRaw(opts.ctx)

	return err
}

func mapKeys(m map[string]bool) []string {
	list := []string{}
	for k := range m {
		list = append(list, k)
	}

	return list
}

func trueMapKeys(m map[string]bool) []string {
	list := []string{}
	for k := range m {
		if m[k] {
			list = append(list, k)
		}
	}

	return list
}
