package main

import (
	"context"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/aquasecurity/fanal/cache"
	fanalTypes "github.com/aquasecurity/fanal/types"
	"github.com/aquasecurity/trivy/pkg/scanner/local"
	trivyTypes "github.com/aquasecurity/trivy/pkg/types"
	"k8s.io/client-go/kubernetes"
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
)
