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

// runtimeSockVolumeName is the CRI hostPath volume name the manager sets on
// Linux pods.
const runtimeSockVolumeName = "runtime-sock-volume"

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
	spec, err := copyAndFillTemplateSpec(newTemplateSpec(), nil, node("linux-node", "linux"), runtimeSpec())
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
	spec, err := copyAndFillTemplateSpec(newTemplateSpec(), nil, node("win-node", "windows"), runtimeSpec())
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
