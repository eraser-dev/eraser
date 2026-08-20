package imagejob

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
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
			{Name: "shared-data"},
		},
		Containers: []corev1.Container{
			{
				Name: "collector",
				VolumeMounts: []corev1.VolumeMount{
					{MountPath: sharedDataMountPath, Name: "shared-data"},
				},
			},
			{
				Name: removerContainer,
				VolumeMounts: []corev1.VolumeMount{
					{MountPath: sharedDataMountPath, Name: "shared-data"},
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
}

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
