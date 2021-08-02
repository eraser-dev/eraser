/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ImageJobSpec defines the desired state of ImageJob
type ImageJobSpec struct {
	// Specifies the job that will be created when executing an ImageJob.
	JobTemplate   v1.PodTemplateSpec `json:"template"`
	ImageListName string             `json:"imageListName"`
}

// ImageJobStatus defines the observed state of ImageJob
type ImageJobStatus struct {
	// Specifies if job was completed, and how many jobs succeeded and failed
	Message string `json:"message"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ImageJob is the Schema for the imagejobs API
type ImageJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageJobSpec   `json:"spec,omitempty"`
	Status ImageJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ImageJobList contains a list of ImageJob
type ImageJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageJob{}, &ImageJobList{})
}
