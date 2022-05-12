package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	machinerytypes "k8s.io/apimachinery/pkg/types"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

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
)

const (
	generalErr = 1

	severityCritical = "CRITICAL"
	severityHigh     = "HIGH"
	severityMedium   = "MEDIUM"
	severityLow      = "LOW"
	severityUnknown  = "UNKNOWN"

	apiPath = "apis/eraser.sh/v1alpha1"
)

var (
	imageListPath = flag.String("image-list", "/etc/images.json", "path to a JSON array of image references")
	cacheDir      = flag.String("cache-dir", "/var/lib/trivy", "path to the cache dir")
	severity      = flag.String("severity", "CRITICAL,HIGH,MEDIUM,LOW,UNKNOWN", "list of severity levels to report")
	ignoreUnfixed = flag.Bool("ignore-unfixed", false, "report only fixed vulnerabilities")

	// Will be modified by parseSeverities() to reflect the `severity` CLI flag
	// These are the only recognized severities and the keys of this map should never be modified.
	severityMap map[string]bool = map[string]bool{
		severityCritical: false,
		severityHigh:     false,
		severityMedium:   false,
		severityLow:      false,
		severityUnknown:  false,
	}
)

type (
	imageList []string

	scannerSetup struct {
		fscache       cache.FSCache
		localScanner  local.Scanner
		scanOptions   trivyTypes.ScanOptions
		dockerOptions fanalTypes.DockerOption
	}

	patch struct {
		Status eraserv1alpha1.ImageCollectorStatus `json:"status"`
	}
)

func init() {
	flag.Parse()

	err := parseSeverity(*severity)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(generalErr)
	}
}

func main() {
	ctx := context.Background()

	cfg, err := rest.InClusterConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(generalErr)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(generalErr)
	}

	result := eraserv1alpha1.ImageCollector{}

	err = clientset.RESTClient().Get().
		AbsPath(apiPath).
		Resource("imagecollectors").
		Name("collector-cr").
		Do(context.Background()).
		Into(&result)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(generalErr)
	}

	scanList := make(map[string]string)
	for _, img := range result.Spec.Images {
		scanList[img.Digest] = img.Name
	}

	err = downloadAndInitDB(*cacheDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(generalErr)
	}

	scanConfig, err := setupScanner(*cacheDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(generalErr)
	}

	resultClient := initializeResultClient()

	imgChan := make(chan string)
	var wg sync.WaitGroup

	for _, imageName := range scanList {
		wg.Add(1)

		go func(imageRef string) {
			fmt.Printf("scanning: %s\n", imageRef)
			dockerImage, cleanup, err := fanalImage.NewDockerImage(ctx, imageRef, scanConfig.dockerOptions)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(generalErr)
			}
			defer cleanup()

			artifactToScan, err := artifactImage.NewArtifact(dockerImage, scanConfig.fscache, artifact.Option{}, config.ScannerOption{})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(generalErr)
			}

			scanner := scanner.NewScanner(scanConfig.localScanner, artifactToScan)

			report, err := scanner.ScanArtifact(ctx, scanConfig.scanOptions)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(generalErr)
			}

		outer:
			for _, result := range report.Results {
				resultClient.FillVulnerabilityInfo(result.Vulnerabilities, result.Type)

				for _, vuln := range result.Vulnerabilities {
					if *ignoreUnfixed && vuln.FixedVersion == "" {
						continue
					}

					if vuln.Severity == "" {
						vuln.Severity = severityUnknown
					}

					if severityMap[vuln.Severity] {
						imgChan <- report.ArtifactName
						break outer
					}
				}
			}

			wg.Done()
		}(imageName)
	}

	go func() {
		wg.Wait()
		close(imgChan)
	}()

	vulnerableImages := make([]eraserv1alpha1.Image, 0, len(scanList))
	for imageRef := range imgChan {
		image := eraserv1alpha1.Image{Digest: "abc123", Name: imageRef}
		vulnerableImages = append(vulnerableImages, image)
	}

	err = updateStatus(ctx, clientset, vulnerableImages)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(generalErr)
	}

	for _, imageRef := range vulnerableImages {
		fmt.Println(imageRef)
	}
}

func parseSeverity(sevString string) error {
	sevs := strings.Split(sevString, ",")
	for _, sev := range sevs {
		_, ok := severityMap[sev]
		if !ok {
			return fmt.Errorf("severity '%s' should be one of of [CRITICAL, HIGH, MEDIUM, LOW, UNKNOWN]", sev)
		}
		severityMap[sev] = true
	}

	return nil
}

func readImageList(imageListPath string) (imageList, error) {
	imageListFile, err := ioutil.ReadFile(imageListPath)
	if err != nil {
		return nil, err
	}

	var scanList imageList
	err = json.Unmarshal(imageListFile, &scanList)
	if err != nil {
		return nil, err
	}

	return scanList, nil
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
	needsUpdate, err := client.NeedsUpdate("dev", false)
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

func setupScanner(cacheDir string) (scannerSetup, error) {
	filesystemCache, err := cache.NewFSCache(cacheDir)
	if err != nil {
		return scannerSetup{}, err
	}

	app := applier.NewApplier(filesystemCache)
	det := ospkg.Detector{}
	dopts := fanalTypes.DockerOption{}
	scan := local.NewScanner(app, det)

	sopts := trivyTypes.ScanOptions{
		VulnType:            []string{"os", "library"},
		SecurityChecks:      []string{"vuln", "secret"},
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

func updateStatus(ctx context.Context, clientset *kubernetes.Clientset, result []eraserv1alpha1.Image) error {
	collectorPatch := patch{
		Status: eraserv1alpha1.ImageCollectorStatus{
			Result: result,
		},
	}

	body, err := json.Marshal(&collectorPatch)
	if err != nil {
		return err
	}

	_, err = clientset.RESTClient().Patch(machinerytypes.MergePatchType).
		AbsPath(apiPath).
		Resource("imagecollectors").
		SubResource("status").
		Name("collector-cr").
		Body(body).DoRaw(ctx)

	return err
}
