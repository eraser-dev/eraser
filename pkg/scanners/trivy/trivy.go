package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	machinerytypes "k8s.io/apimachinery/pkg/types"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/Azure/eraser/pkg/logger"
	"github.com/aquasecurity/fanal/analyzer/config"
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
	severity        = flag.String("severity", "CRITICAL,HIGH", "list of severity levels to report")
	vulnTypes       = flag.String("vuln-type", "os,library", "comma separated list of vulnerability types")
	securityChecks  = flag.String("security-checks", "vuln,secret", "comma-separated list of what security issues to detect")
	ignoreUnfixed   = flag.Bool("ignore-unfixed", true, "report only fixed vulnerabilities")

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
		clientset        *kubernetes.Clientset
		collectorCRName  string
		resourceName     string
		subResourceName  string
		vulnerableImages []eraserv1alpha1.Image
		failedImages     []eraserv1alpha1.Image
	}

	imageReport struct {
		eraserv1alpha1.Image
		err error
	}
)

func main() {
	flag.Parse()
	ctx := context.Background()

	if err := logger.Configure(); err != nil {
		fmt.Fprintln(os.Stderr, "Error setting up logger:", err)
		os.Exit(generalErr)
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
	dbMutex := sync.Mutex{}
	imgChan := make(chan imageReport)
	var wg sync.WaitGroup

	for k := range result.Spec.Images {
		img := result.Spec.Images[k]
		wg.Add(1)

		go func(img eraserv1alpha1.Image) {
			imageRef := img.Name
			if imageRef == "" {
				log.Info("found image with no name", "img", img)
				imgChan <- imageReport{Image: img, err: fmt.Errorf("no name")}
				wg.Done()
				return
			}

			log.Info("scanning image", "imageRef", imageRef)

			// Reads db and cache and potentially modifies them, so lock all
			// other threads during this operation
			dbMutex.Lock()
			dockerImage, cleanup, err := fanalImage.NewDockerImage(ctx, imageRef, scanConfig.dockerOptions)
			defer cleanup()
			dbMutex.Unlock()

			if err != nil {
				imgChan <- imageReport{Image: img, err: err}
				log.Error(err, "error fetching manifest for image", "img", img)
				wg.Done()
				return
			}

			artifactToScan, err := artifactImage.NewArtifact(dockerImage, scanConfig.fscache, artifact.Option{}, config.ScannerOption{})
			if err != nil {
				imgChan <- imageReport{Image: img, err: err}
				log.Error(err, "error registering config for artifact", "img", img)
				wg.Done()
				return
			}

			scanner := scanner.NewScanner(scanConfig.localScanner, artifactToScan)
			report, err := scanner.ScanArtifact(ctx, scanConfig.scanOptions)
			if err != nil {
				imgChan <- imageReport{Image: img, err: err}
				log.Error(err, "error scanning image", "img", img)
				wg.Done()
				return
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
						imgChan <- imageReport{Image: img, err: nil}
						break outer
					}
				}
			}

			wg.Done()
		}(img)
	}

	go func() {
		wg.Wait()
		close(imgChan)
	}()

	vulnerableImages := make([]eraserv1alpha1.Image, 0, len(result.Spec.Images))
	failedImages := make([]eraserv1alpha1.Image, 0, len(result.Spec.Images))
	for img := range imgChan {
		if img.err != nil {
			log.V(1).Error(img.err, "error scanning image") // debug only
			failedImages = append(failedImages, img.Image)
			log.Info("unable to scan image", "img", img)
			continue
		}

		log.Info("image has vulnerabilities", "img", img)
		vulnerableImages = append(vulnerableImages, img.Image)
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
