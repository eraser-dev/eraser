package utils

import (
	corev1 "k8s.io/api/core/v1"
)

var trueval = true

var SharedSecurityContext = &corev1.SecurityContext{
	Capabilities: &corev1.Capabilities{
		Drop: []corev1.Capability{"ALL"},
	},
	ReadOnlyRootFilesystem: &trueval,
	SeccompProfile: &corev1.SeccompProfile{
		Type: corev1.SeccompProfileTypeRuntimeDefault,
	},
}

// WindowsHostProcessUserName is the built-in account HostProcess worker pods run
// as so they can reach the containerd named pipe on a Windows node.
const WindowsHostProcessUserName = `NT AUTHORITY\SYSTEM`

// WindowsHostProcessPodSecurityContext returns the pod-level security context
// for a Windows worker pod. It runs the pod as a HostProcess pod under
// NT AUTHORITY\SYSTEM. This is the Windows-safe replacement for the Linux-only
// SharedSecurityContext, whose capabilities, seccompProfile and
// readOnlyRootFilesystem fields are not valid on Windows.
func WindowsHostProcessPodSecurityContext() *corev1.PodSecurityContext {
	hostProcess := true
	userName := WindowsHostProcessUserName
	return &corev1.PodSecurityContext{
		WindowsOptions: &corev1.WindowsSecurityContextOptions{
			HostProcess:   &hostProcess,
			RunAsUserName: &userName,
		},
	}
}
