package imagejob

import (
	"context"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// startEnvtest brings up a real kube-apiserver + etcd so that emitted pod specs
// can be checked against the apiserver's own validation (dry-run create). It is
// skipped when the envtest binaries are not available (KUBEBUILDER_ASSETS unset),
// so `go test` still works without them; `make test` sets that variable.
func startEnvtest(t *testing.T) (client.Client, func()) {
	t.Helper()

	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("KUBEBUILDER_ASSETS not set; skipping apiserver validation test (run via `make test`)")
	}

	testEnv := &envtest.Environment{}
	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("failed to start envtest: %v", err)
	}

	cl, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		_ = testEnv.Stop()
		t.Fatalf("failed to build client: %v", err)
	}

	return cl, func() { _ = testEnv.Stop() }
}

func windowsPod(spec *corev1.PodSpec) *corev1.Pod {
	// NodeName is set by the manager, but a dry-run create against the apiserver
	// should validate the spec without a real node; clear it to avoid binding.
	s := spec.DeepCopy()
	s.NodeName = ""

	// Fill fields the manager does not touch but the apiserver requires, so the
	// test exercises OS validation rather than unrelated required-field errors.
	backed := map[string]bool{}
	for i := range s.Volumes {
		backed[s.Volumes[i].Name] = true
	}
	for i := range s.Containers {
		if s.Containers[i].Image == "" {
			s.Containers[i].Image = "example.invalid/image:latest"
		}
		for _, vm := range s.Containers[i].VolumeMounts {
			if vm.Name == "" || backed[vm.Name] {
				continue
			}
			s.Volumes = append(s.Volumes, corev1.Volume{
				Name:         vm.Name,
				VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			})
			backed[vm.Name] = true
		}
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "eraser-win-validate-",
			Namespace:    "default",
		},
		Spec: *s,
	}
}

// TestWindowsPodSpecPassesAPIServerValidation is a guardrail: the Windows pod
// spec the manager emits must be accepted by the Kubernetes apiserver. Because
// the spec declares spec.os.name=windows, the apiserver runs validateOSFields,
// which rejects Linux-only fields. If a future change adds a Windows-unsupported
// field to the emitted spec, this test fails at admission rather than the break
// only surfacing on a live Windows node.
func TestWindowsPodSpecPassesAPIServerValidation(t *testing.T) {
	cl, stop := startEnvtest(t)
	defer stop()

	spec, err := copyAndFillTemplateSpec(newTemplateSpec(), nil, node("win-node", "windows"), runtimeSpec())
	if err != nil {
		t.Fatalf("copyAndFillTemplateSpec: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pod := windowsPod(spec)
	if err := cl.Create(ctx, pod, client.DryRunAll); err != nil {
		t.Fatalf("apiserver rejected the emitted Windows pod spec: %v", err)
	}
}

// TestWindowsPodSpecRejectsLinuxOnlyField proves the guardrail has teeth: a
// Windows pod carrying a Linux-only container field (capabilities, exactly what
// SharedSecurityContext sets) must be rejected by the apiserver. This is what
// catches a regression where someone forgets to clear a Linux-only field on the
// Windows path.
func TestWindowsPodSpecRejectsLinuxOnlyField(t *testing.T) {
	cl, stop := startEnvtest(t)
	defer stop()

	spec, err := copyAndFillTemplateSpec(newTemplateSpec(), nil, node("win-node", "windows"), runtimeSpec())
	if err != nil {
		t.Fatalf("copyAndFillTemplateSpec: %v", err)
	}

	// Re-introduce a Linux-only field on a container, simulating a regression.
	spec.Containers[0].SecurityContext = &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = cl.Create(ctx, windowsPod(spec), client.DryRunAll)
	if err == nil {
		t.Fatal("expected apiserver to reject a Windows pod with a Linux-only capabilities field, but it was accepted")
	}
	if !apierrors.IsInvalid(err) {
		t.Fatalf("expected an Invalid validation error, got: %v", err)
	}
}
