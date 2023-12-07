package main

import (
	"os"

	unversioned "github.com/eraser-dev/eraser/api/unversioned"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func loadConfig(filename string) (Config, error) {
	cfg := *DefaultConfig()

	b, err := os.ReadFile(filename)
	if err != nil {
		log.Info("LOADCONFIG unable to read file")
		return cfg, err
	}

	var eraserConfig unversioned.EraserConfig
	err = yaml.Unmarshal(b, &eraserConfig)
	if err != nil {
		log.Info("LOADCONFIG yaml.Unmarshal error")
		return cfg, err
	}

	scanCfgYaml := eraserConfig.Components.Scanner.Config
	scanCfgBytes := []byte("")
	if scanCfgYaml != nil {
		scanCfgBytes = []byte(*scanCfgYaml)
	}

	err = yaml.Unmarshal(scanCfgBytes, &cfg)
	if err != nil {
		log.Info("LOADCONFIG scannner config yaml.Unmarshal error")
		return cfg, err
	}

	return cfg, nil
}
