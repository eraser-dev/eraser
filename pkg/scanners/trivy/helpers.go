package main

import (
	"os"

	unversioned "github.com/eraser-dev/eraser/api/unversioned"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func loadConfig(filename string) (Config, error) {
	var eraserConfig unversioned.EraserConfig
	cfg := *DefaultConfig()

	b, err := os.ReadFile(filename)
	if err != nil {
		return cfg, err
	}

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

	cfg.RuntimeAddress = string(eraserConfig.Manager.RuntimeSocketAddress)

	return cfg, nil
}
