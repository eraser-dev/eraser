package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = schema.GroupVersion{
	Group:   "eraser.sh",
	Version: "v1alpha1",
}

var (
	localSchemeBuilder = &SchemeBuilder
)

func init() {
	SchemeBuilder.Register(addKnownTypes)
}

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&ImageCollector{},
		&ImageCollectorList{},
		&ImageList{},
		&ImageListList{},
		&ImageJob{},
		&ImageJobList{},
	)
	scheme.AddKnownTypes(SchemeGroupVersion, &metav1.Status{})
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)

	return nil
}
