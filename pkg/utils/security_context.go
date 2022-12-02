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
