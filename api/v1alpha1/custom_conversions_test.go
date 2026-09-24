package v1alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"

	unversioned "github.com/eraser-dev/eraser/api/unversioned"
)

func TestConvertOptionalContainerConfigToScannerConfig(t *testing.T) {
	in := OptionalContainerConfig{
		Enabled: true,
		ContainerConfig: ContainerConfig{
			Image:   RepoTag{Repo: "scanner", Tag: "v1"},
			Request: ResourceRequirements{Mem: resource.MustParse("100Mi"), CPU: resource.MustParse("50m")},
			Limit:   ResourceRequirements{Mem: resource.MustParse("1Gi")},
		},
	}

	var out unversioned.ScannerConfig
	if err := Convert_v1alpha1_OptionalContainerConfig_To_unversioned_ScannerConfig(&in, &out, nil); err != nil {
		t.Fatalf("convert: %v", err)
	}

	if !out.Enabled {
		t.Error("Enabled was not carried across")
	}
	if got, want := out.Image.Repo, "scanner"; got != want {
		t.Errorf("Image.Repo = %q, want %q", got, want)
	}
	if got, want := out.Request.Mem.String(), "100Mi"; got != want {
		t.Errorf("Request.Mem = %q, want %q", got, want)
	}
	if out.Windows != nil {
		t.Error("Windows must be nil: v1alpha1 has no per-OS override to convert from")
	}
}

func TestConvertScannerConfigToOptionalContainerConfigDropsWindows(t *testing.T) {
	in := unversioned.ScannerConfig{
		OptionalContainerConfig: unversioned.OptionalContainerConfig{
			Enabled: true,
			ContainerConfig: unversioned.ContainerConfig{
				Image: unversioned.RepoTag{Repo: "linux-scanner", Tag: "v1"},
			},
		},
		Windows: &unversioned.WindowsScannerConfig{
			Image: unversioned.RepoTag{Repo: "windows-scanner", Tag: "v2"},
		},
	}

	var out OptionalContainerConfig
	if err := Convert_unversioned_ScannerConfig_To_v1alpha1_OptionalContainerConfig(&in, &out, nil); err != nil {
		t.Fatalf("convert: %v", err)
	}

	// v1alpha1 cannot express the override, so the Linux side must survive
	// intact and the Windows side must simply be dropped.
	if got, want := out.Image.Repo, "linux-scanner"; got != want {
		t.Errorf("Image.Repo = %q, want %q", got, want)
	}
	if !out.Enabled {
		t.Error("Enabled was not carried across")
	}
}
