package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/aquasecurity/trivy-db/pkg/db"
	dlDb "github.com/aquasecurity/trivy/pkg/db"
	"github.com/aquasecurity/trivy/pkg/detector/ospkg"
	"github.com/aquasecurity/trivy/pkg/fanal/applier"
	"github.com/aquasecurity/trivy/pkg/fanal/cache"
	fanalTypes "github.com/aquasecurity/trivy/pkg/fanal/types"
	pkgResult "github.com/aquasecurity/trivy/pkg/result"
	"github.com/aquasecurity/trivy/pkg/scanner/local"
	trivyTypes "github.com/aquasecurity/trivy/pkg/types"
)

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
	client := dlDb.NewClient(cacheDir, true, true)
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
