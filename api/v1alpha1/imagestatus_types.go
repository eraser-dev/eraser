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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImageStatusSpec defines the desired state of ImageStatus
type ImageStatusSpec struct {
}

// ImageStatusStatus defines the observed state of ImageStatus
type ImageStatusStatus struct {
	// Specifies if the image removal was a "success" or "error"
	Status string `json:"status"`

	// Message for reason for error, if applicable.
	// +optional
	Message string `json:"message"`

	// Specifies on which node image removal took place.
	Node string `json:"node"`

	// Specifies name of vulnerable image.
	Name string `json:"name"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ImageStatus is the Schema for the imagestatus API
type ImageStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status ImageStatusStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ImageStatusList contains a list of ImageStatus
type ImageStatusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageJobStatus `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageStatus{}, &ImageStatus{})
}
