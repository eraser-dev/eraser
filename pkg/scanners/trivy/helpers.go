package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/aquasecurity/trivy-db/pkg/db"
	dlDb "github.com/aquasecurity/trivy/pkg/db"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func loadConfig(filename string) (Config, error) {
	cfg := *DefaultConfig()

	b, err := os.ReadFile(filename)
	if err != nil {
		return cfg, err
	}

	var eraserConfig eraserv1alpha1.EraserConfig
	err = yaml.Unmarshal(b, &eraserConfig)
	if err != nil {
		return cfg, err
	}

	scanCfgYaml := eraserConfig.Components.Scanner.Config
	scanCfgBytes := []byte("")
	if scanCfgYaml != nil {
		scanCfgBytes = []byte(*scanCfgYaml)
	}

	err = yaml.Unmarshal(scanCfgBytes, &cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
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

func downloadAndInitDB(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("valid configuration required")
	}

	err := downloadDB(cfg)
	if err != nil {
		return err
	}

	err = db.Init(cfg.CacheDir)
	if err != nil {
		return err
	}

	return nil
}

func downloadDB(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("valid configuration required")
	}

	client := dlDb.NewClient(cfg.CacheDir, true, true, dlDb.WithDBRepository(cfg.DBRepo))
	ctx := context.Background()
	needsUpdate, err := client.NeedsUpdate(trivyVersion, false)
	if err != nil {
		return err
	}

	if needsUpdate {
		if err = client.Download(ctx, cfg.CacheDir); err != nil {
			return err
		}
	}

	return nil
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
