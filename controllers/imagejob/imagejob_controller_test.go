package imagejob

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eraser-dev/eraser/api/unversioned"
	controllerUtils "github.com/eraser-dev/eraser/controllers/util"
	eraserUtils "github.com/eraser-dev/eraser/pkg/utils"
)

// sharedDataMountPath mirrors the Linux path the collector controller bakes into
// the pod template before the manager fills it per node.
const sharedDataMountPath = eraserUtils.LinuxSharedDataPath

func newTemplateSpec() *corev1.PodSpec {
	return &corev1.PodSpec{
		Volumes: []corev1.Volume{
			{Name: "shared-data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		},
		Containers: []corev1.Container{
			{
				Name: "collector",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("25Mi")},
					Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("30Mi")},
				},
				VolumeMounts: []corev1.VolumeMount{
					{MountPath: sharedDataMountPath, Name: "shared-data"},
				},
			},
			{
				Name: removerContainer,
				Args: []string{"--imagelist=/run/eraser.sh/imagelist/images", "--log-level=info"},
				VolumeMounts: []corev1.VolumeMount{
					{MountPath: sharedDataMountPath, Name: "shared-data"},
					{MountPath: "/run/eraser.sh/imagelist", Name: "imagelist"},
				},
				SecurityContext: eraserUtils.SharedSecurityContext,
			},
		},
	}
}

func node(name, osLabel string) *corev1.Node {
	n := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if osLabel != "" {
		n.Labels = map[string]string{corev1.LabelOSStable: osLabel}
	}
	return n
}

func runtimeSpec() *unversioned.RuntimeSpec {
	return &unversioned.RuntimeSpec{
		Name:    unversioned.RuntimeContainerd,
		Address: "unix:///run/containerd/containerd.sock",
	}
}

func hasVolume(spec *corev1.PodSpec, name string) bool {
	for i := range spec.Volumes {
		if spec.Volumes[i].Name == name {
			return true
		}
	}
	return false
}

func containerMountPath(c *corev1.Container, volumeName string) (string, bool) {
	for i := range c.VolumeMounts {
		if c.VolumeMounts[i].Name == volumeName {
			return c.VolumeMounts[i].MountPath, true
		}
	}
	return "", false
}

func TestCopyAndFillTemplateSpecLinux(t *testing.T) {
	spec, err := copyAndFillTemplateSpec(newTemplateSpec(), nil, node("linux-node", "linux"), runtimeSpec(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if spec.NodeName != "linux-node" {
		t.Errorf("NodeName = %q, want linux-node", spec.NodeName)
	}
	if spec.HostNetwork {
		t.Error("HostNetwork should be false for a Linux node")
	}
	if spec.OS != nil {
		t.Errorf("spec.OS should be unset on a Linux node, got %v", spec.OS)
	}
	if !hasVolume(spec, runtimeSockVolumeName) {
		t.Errorf("expected CRI hostPath volume %q on a Linux node", runtimeSockVolumeName)
	}
	for i := range spec.Containers {
		if p, ok := containerMountPath(&spec.Containers[i], runtimeSockVolumeName); !ok {
			t.Errorf("container %q missing CRI mount", spec.Containers[i].Name)
		} else if p != controllerUtils.CRIPath {
			t.Errorf("CRI mount path = %q, want %q", p, controllerUtils.CRIPath)
		}
		if p, _ := containerMountPath(&spec.Containers[i], "shared-data"); p != eraserUtils.LinuxSharedDataPath {
			t.Errorf("shared-data mount = %q, want Linux path %q", p, eraserUtils.LinuxSharedDataPath)
		}
	}
	// Linux security context is preserved on the remover container.
	sc := spec.Containers[1].SecurityContext
	if sc == nil || sc.ReadOnlyRootFilesystem == nil || !*sc.ReadOnlyRootFilesystem {
		t.Error("expected Linux SharedSecurityContext preserved on Linux node")
	}
}

func TestCopyAndFillTemplateSpecWindows(t *testing.T) {
	spec, err := copyAndFillTemplateSpec(newTemplateSpec(), nil, node("win-node", "windows"), runtimeSpec(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if spec.NodeName != "win-node" {
		t.Errorf("NodeName = %q, want win-node", spec.NodeName)
	}
	if !spec.HostNetwork {
		t.Error("HostNetwork should be true for a Windows HostProcess pod")
	}
	if spec.OS == nil || spec.OS.Name != corev1.Windows {
		t.Errorf("spec.OS = %v, want Name=windows", spec.OS)
	}
	if hasVolume(spec, runtimeSockVolumeName) {
		t.Error("CRI hostPath volume must be omitted on a Windows node")
	}

	if spec.SecurityContext == nil || spec.SecurityContext.WindowsOptions == nil {
		t.Fatal("expected pod-level Windows HostProcess security context")
	}
	wo := spec.SecurityContext.WindowsOptions
	if wo.HostProcess == nil || !*wo.HostProcess {
		t.Error("HostProcess should be true")
	}
	if wo.RunAsUserName == nil || *wo.RunAsUserName != eraserUtils.WindowsHostProcessUserName {
		t.Errorf("RunAsUserName = %v, want %q", wo.RunAsUserName, eraserUtils.WindowsHostProcessUserName)
	}

	for i := range spec.Containers {
		c := &spec.Containers[i]
		if _, ok := containerMountPath(c, runtimeSockVolumeName); ok {
			t.Errorf("container %q should not have a CRI mount on Windows", c.Name)
		}
		if p, _ := containerMountPath(c, "shared-data"); p != eraserUtils.WindowsSharedDataPath {
			t.Errorf("shared-data mount = %q, want Windows path %q", p, eraserUtils.WindowsSharedDataPath)
		}
		if c.SecurityContext != nil {
			t.Errorf("container %q Linux SecurityContext should be cleared on Windows", c.Name)
		}
	}

	// collector had a 30Mi memory limit (< 256Mi) -> raised to 256Mi; its 25Mi
	// request is preserved.
	col := &spec.Containers[0]
	wantMin := resource.MustParse("256Mi")
	if got := col.Resources.Limits[corev1.ResourceMemory]; got.Cmp(wantMin) != 0 {
		t.Errorf("collector memory limit = %s, want 256Mi", got.String())
	}
	if got := col.Resources.Requests[corev1.ResourceMemory]; got.Cmp(resource.MustParse("25Mi")) != 0 {
		t.Errorf("collector memory request = %s, want 25Mi (unchanged)", got.String())
	}

	// remover had no memory limit -> left unset (no Job Object memory cap).
	rmv := &spec.Containers[1]
	if _, ok := rmv.Resources.Limits[corev1.ResourceMemory]; ok {
		t.Error("remover had no memory limit; it should stay unset on Windows")
	}

	// The remover's imagelist mount and --imagelist arg must be rewritten to Windows paths.
	rem := &spec.Containers[1]
	if p, _ := containerMountPath(rem, "imagelist"); p != `C:\run\eraser.sh\imagelist` {
		t.Errorf("imagelist mount = %q, want C:\\run\\eraser.sh\\imagelist", p)
	}
	if rem.Args[0] != `--imagelist=C:\run\eraser.sh\imagelist\images` {
		t.Errorf("imagelist arg = %q, want windows form", rem.Args[0])
	}
	if rem.Args[1] != "--log-level=info" {
		t.Errorf("non-path arg should be untouched, got %q", rem.Args[1])
	}
	// HostProcess containers need an explicit sandbox-relative command.
	wantCmd := `%CONTAINER_SANDBOX_MOUNT_POINT%\remover.exe`
	if len(rem.Command) != 1 || rem.Command[0] != wantCmd {
		t.Errorf("remover command = %v, want [%s]", rem.Command, wantCmd)
	}
}

func TestSkipWindowsScannerNodes(t *testing.T) {
	newNodes := func() []corev1.Node {
		return []corev1.Node{
			*node("linux-node", "linux"),
			*node("win-node", "windows"),
		}
	}

	t.Run("scanner enabled skips windows node", func(t *testing.T) {
		kept, skipped := skipWindowsScannerNodes(newNodes(), 0, true, false)
		if skipped != 1 {
			t.Errorf("skipped = %d, want 1 (the windows node)", skipped)
		}
		if len(kept) != 1 || kept[0].Name != "linux-node" {
			names := make([]string, len(kept))
			for i := range kept {
				names[i] = kept[i].Name
			}
			t.Errorf("kept nodes = %v, want [linux-node]", names)
		}
	})

	t.Run("scanner enabled preserves prior skipped count", func(t *testing.T) {
		_, skipped := skipWindowsScannerNodes(newNodes(), 3, true, false)
		if skipped != 4 {
			t.Errorf("skipped = %d, want 4 (3 prior + 1 windows)", skipped)
		}
	})

	t.Run("scanner disabled keeps all nodes", func(t *testing.T) {
		kept, skipped := skipWindowsScannerNodes(newNodes(), 0, false, false)
		if len(kept) != 2 || skipped != 0 {
			t.Errorf("got kept=%d skipped=%d, want both nodes kept", len(kept), skipped)
		}
	})

	t.Run("windows scanner configured keeps windows node", func(t *testing.T) {
		kept, skipped := skipWindowsScannerNodes(newNodes(), 0, true, true)
		if len(kept) != 2 || skipped != 0 {
			t.Errorf("got kept=%d skipped=%d, want both nodes kept when a windows scanner is configured", len(kept), skipped)
		}
	})
}

// newCollectorTemplateSpec mirrors the collector job template, where the
// scanner is appended last (index 2).
func newCollectorTemplateSpec(scannerImage string) *corev1.PodSpec {
	spec := newTemplateSpec()
	spec.Containers = append(spec.Containers, corev1.Container{
		Name:  "trivy-scanner",
		Image: scannerImage,
		VolumeMounts: []corev1.VolumeMount{
			{MountPath: sharedDataMountPath, Name: "shared-data"},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("500Mi"),
				corev1.ResourceCPU:    resource.MustParse("1000m"),
			},
			Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("2Gi")},
		},
	})
	return spec
}

// A windows block that names no image must not act as the opt-in: it would admit
// Windows nodes and then hand them the invalid image ":".
func TestUsableWindowsScannerRejectsBlocksWithoutAnImage(t *testing.T) {
	rejected := map[string]*unversioned.WindowsScannerConfig{
		"nil":            nil,
		"zero value":     {},
		"resources only": {Request: unversioned.ResourceRequirements{Mem: resource.MustParse("200Mi")}},
		"repo only":      {Image: unversioned.RepoTag{Repo: "scanner"}},
		"tag only":       {Image: unversioned.RepoTag{Tag: "v1"}},
	}

	for name, w := range rejected {
		t.Run(name, func(t *testing.T) {
			if got := usableWindowsScanner(&unversioned.ScannerConfig{Windows: w}); got != nil {
				t.Errorf("usableWindowsScanner = %+v, want nil", got)
			}
		})
	}

	t.Run("complete image is accepted", func(t *testing.T) {
		want := &unversioned.WindowsScannerConfig{Image: unversioned.RepoTag{Repo: "scanner", Tag: "v1"}}
		if got := usableWindowsScanner(&unversioned.ScannerConfig{Windows: want}); got != want {
			t.Errorf("usableWindowsScanner = %+v, want the configured override", got)
		}
	})
}

// One ImageJob template plus one config must yield trivy on Linux and the
// Windows-capable scanner on Windows.
func TestPerOSScannerImage(t *testing.T) {
	const (
		linuxScanner = "ghcr.io/eraser-dev/eraser-trivy-scanner:v1.5.0-beta.1"
		winScanner   = "ghcr.io/charleswool/eraser-fake-scanner:v0.2.2"
	)

	winOverride := &unversioned.WindowsScannerConfig{
		Image:   unversioned.RepoTag{Repo: "ghcr.io/charleswool/eraser-fake-scanner", Tag: "v0.2.2"},
		Request: unversioned.ResourceRequirements{Mem: resource.MustParse("200Mi"), CPU: resource.MustParse("500m")},
		Limit:   unversioned.ResourceRequirements{Mem: resource.MustParse("1Gi")},
	}

	linuxSpec, err := copyAndFillTemplateSpec(newCollectorTemplateSpec(linuxScanner), nil, node("linux-node", "linux"), runtimeSpec(), winOverride)
	if err != nil {
		t.Fatalf("linux node: %v", err)
	}
	winSpec, err := copyAndFillTemplateSpec(newCollectorTemplateSpec(linuxScanner), nil, node("win-node", "windows"), runtimeSpec(), winOverride)
	if err != nil {
		t.Fatalf("windows node: %v", err)
	}

	if got := linuxSpec.Containers[scannerContainerIdx].Image; got != linuxScanner {
		t.Errorf("linux scanner image = %q, want %q", got, linuxScanner)
	}
	if got := winSpec.Containers[scannerContainerIdx].Image; got != winScanner {
		t.Errorf("windows scanner image = %q, want %q", got, winScanner)
	}

	// Only the scanner may diverge; the other components ship one multi-platform index.
	for i := 0; i < scannerContainerIdx; i++ {
		if linuxSpec.Containers[i].Image != winSpec.Containers[i].Image {
			t.Errorf("container %q image diverged: linux=%q windows=%q",
				linuxSpec.Containers[i].Name, linuxSpec.Containers[i].Image, winSpec.Containers[i].Image)
		}
	}

	if got := winSpec.Containers[scannerContainerIdx].Resources.Requests.Cpu().String(); got != "500m" {
		t.Errorf("windows scanner cpu request = %q, want 500m from the override", got)
	}
	if got := linuxSpec.Containers[scannerContainerIdx].Resources.Requests.Cpu().String(); got != "1" {
		t.Errorf("linux scanner cpu request = %q, want 1 from the template", got)
	}
}

// Without an override the Windows scanner container is left untouched, so the
// node-skip path remains the only thing keeping a Linux-only scanner off Windows.
func TestPerOSScannerImageAbsentOverride(t *testing.T) {
	const linuxScanner = "ghcr.io/eraser-dev/eraser-trivy-scanner:v1.5.0-beta.1"

	winSpec, err := copyAndFillTemplateSpec(newCollectorTemplateSpec(linuxScanner), nil, node("win-node", "windows"), runtimeSpec(), nil)
	if err != nil {
		t.Fatalf("windows node: %v", err)
	}
	if got := winSpec.Containers[scannerContainerIdx].Image; got != linuxScanner {
		t.Errorf("scanner image = %q, want it unchanged at %q", got, linuxScanner)
	}
}

// A request-only override must not emit limits.memory: 0, which sits below the
// request and gets the pod rejected.
func TestPerOSScannerRequestOnlyOverrideOmitsMemoryLimit(t *testing.T) {
	const linuxScanner = "ghcr.io/eraser-dev/eraser-trivy-scanner:v1.5.0-beta.1"

	winOverride := &unversioned.WindowsScannerConfig{
		Image:   unversioned.RepoTag{Repo: "example.com/win-scanner", Tag: "v1"},
		Request: unversioned.ResourceRequirements{Mem: resource.MustParse("200Mi")},
	}

	winSpec, err := copyAndFillTemplateSpec(newCollectorTemplateSpec(linuxScanner), nil, node("win-node", "windows"), runtimeSpec(), winOverride)
	if err != nil {
		t.Fatalf("windows node: %v", err)
	}

	scanner := winSpec.Containers[scannerContainerIdx]
	if _, ok := scanner.Resources.Limits[corev1.ResourceMemory]; ok {
		t.Errorf("memory limit = %q, want no entry when none is configured",
			scanner.Resources.Limits.Memory().String())
	}
	if got := scanner.Resources.Requests.Memory().String(); got != "200Mi" {
		t.Errorf("memory request = %q, want 200Mi", got)
	}
}

// The windows exclusion has to go before node filtering runs, and the selector
// slice comes straight from the parsed config, so it must not be edited in place.
func TestWithoutWindowsFilterLabel(t *testing.T) {
	in := []string{defaultFilterLabel, windowsFilterLabel, "example.com/other"}

	got := withoutWindowsFilterLabel(in)
	want := []string{defaultFilterLabel, "example.com/other"}

	if len(got) != len(want) {
		t.Fatalf("selectors = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("selectors = %v, want %v", got, want)
		}
	}

	if len(in) != 3 || in[1] != windowsFilterLabel {
		t.Errorf("input slice was mutated: %v", in)
	}

	// Nothing to drop is not an error, and the default label survives.
	if got := withoutWindowsFilterLabel([]string{defaultFilterLabel}); len(got) != 1 || got[0] != defaultFilterLabel {
		t.Errorf("selectors = %v, want [%s]", got, defaultFilterLabel)
	}
}

func TestFillWindowsPodSpecDisablesTelemetry(t *testing.T) {
	spec := newTemplateSpec()
	// The manager injects the OTLP endpoint on worker containers; here the
	// remover container carries a cluster service-style endpoint.
	spec.Containers[1].Env = append(spec.Containers[1].Env, corev1.EnvVar{
		Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
		Value: "otel-collector:4318",
	})

	fillWindowsPodSpec(spec, nil)

	found := false
	for i := range spec.Containers {
		for _, e := range spec.Containers[i].Env {
			if e.Name == "OTEL_EXPORTER_OTLP_ENDPOINT" {
				found = true
				if e.Value != "" {
					t.Errorf("container %q OTLP endpoint = %q, want cleared on windows", spec.Containers[i].Name, e.Value)
				}
			}
		}
	}
	if !found {
		t.Error("expected the OTLP endpoint env var to be present (and cleared), but it was missing")
	}
}

func TestRaiseWindowsMemoryLimit(t *testing.T) {
	cases := []struct {
		name    string
		limit   *string // nil = unset
		request *string // nil = unset
		want    *string // nil = still unset
	}{
		{"below min is raised", ptr("30Mi"), nil, ptr("256Mi")},
		{"at min is unchanged", ptr("256Mi"), nil, ptr("256Mi")},
		{"above min is unchanged", ptr("512Mi"), nil, ptr("512Mi")},
		{"explicit zero stays unlimited", ptr("0"), nil, ptr("0")},
		{"unset stays unset", nil, nil, nil},
		{"floor is at least the request", ptr("30Mi"), ptr("300Mi"), ptr("300Mi")},
		{"request below min still floors at min", ptr("30Mi"), ptr("100Mi"), ptr("256Mi")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &corev1.Container{}
			if tc.limit != nil {
				c.Resources.Limits = corev1.ResourceList{corev1.ResourceMemory: resource.MustParse(*tc.limit)}
			}
			if tc.request != nil {
				c.Resources.Requests = corev1.ResourceList{corev1.ResourceMemory: resource.MustParse(*tc.request)}
			}
			raiseWindowsMemoryLimit(c)
			got, ok := c.Resources.Limits[corev1.ResourceMemory]
			if tc.want == nil {
				if ok {
					t.Errorf("memory limit = %s, want unset", got.String())
				}
				return
			}
			if !ok || got.Cmp(resource.MustParse(*tc.want)) != 0 {
				t.Errorf("memory limit = %v, want %s", got, *tc.want)
			}
		})
	}
}

func ptr(s string) *string { return &s }

func TestIsWindowsNode(t *testing.T) {
	cases := []struct {
		name string
		node *corev1.Node
		want bool
	}{
		{"label windows", node("a", "windows"), true},
		{"label linux", node("b", "linux"), false},
		{"no label falls back to nodeinfo", &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "c"},
			Status:     corev1.NodeStatus{NodeInfo: corev1.NodeSystemInfo{OperatingSystem: "windows"}},
		}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isWindowsNode(tc.node); got != tc.want {
				t.Errorf("isWindowsNode = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLinuxToWindowsEraserPath(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"exact eraser path", "/run/eraser.sh", `C:\run\eraser.sh`},
		{"file under eraser dir", "/run/eraser.sh/imagelist/images", `C:\run\eraser.sh\imagelist\images`},
		{"trailing slash", "/run/eraser.sh/", `C:\run\eraser.sh\`},
		// Sibling paths that only share the textual prefix must be left intact.
		{"sibling dir suffix", "/run/eraser.sh-old/imagelist", "/run/eraser.sh-old/imagelist"},
		{"sibling file suffix", "/run/eraser.shx", "/run/eraser.shx"},
		{"unrelated path", "/var/lib/foo", "/var/lib/foo"},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := linuxToWindowsEraserPath(tc.in); got != tc.want {
				t.Errorf("linuxToWindowsEraserPath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestTranslateEraserArg(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"bare path", "/run/eraser.sh/imagelist/images", `C:\run\eraser.sh\imagelist\images`},
		{"flag with path", "--imagelist=/run/eraser.sh/imagelist/images", `--imagelist=C:\run\eraser.sh\imagelist\images`},
		{"flag with exact path", "--dir=/run/eraser.sh", `--dir=C:\run\eraser.sh`},
		{"non-path arg untouched", "--log-level=info", "--log-level=info"},
		// Sibling path embedded in an arg must not be rewritten.
		{"sibling path in flag", "--imagelist=/run/eraser.sh-old/imagelist", "--imagelist=/run/eraser.sh-old/imagelist"},
		{"sibling file suffix", "--path=/run/eraser.shx", "--path=/run/eraser.shx"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := translateEraserArg(tc.in); got != tc.want {
				t.Errorf("translateEraserArg(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
