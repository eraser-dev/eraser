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
		log.Error(err, "unable to read eraser config")
		return cfg, err
	}

	var eraserConfig unversioned.EraserConfig
	err = yaml.Unmarshal(b, &eraserConfig)
	if err != nil {
		log.Error(err, "unable to unmarshal eraser config")
	}

	scanCfgYaml := eraserConfig.Components.Scanner.Config
	scanCfgBytes := []byte("")
	if scanCfgYaml != nil {
		scanCfgBytes = []byte(*scanCfgYaml)
	}

	err = yaml.Unmarshal(scanCfgBytes, &cfg)
	if err != nil {
		log.Error(err, "unable to unmarshal scanner config")
		return cfg, err
	}

	return cfg, nil
}
