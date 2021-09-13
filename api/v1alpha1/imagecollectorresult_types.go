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

// ImageCollectorResultStatus defines the observed state of ImageCollectorResult
type ImageCollectorResultStatus struct {
	// Specifies on which node the listing operation took place
	Node string `json:"node"`
	// The list of images from the node
	ImagesResults []string `json:"images"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope="Cluster"
// ImageCollectorResult is the Schema for the imagecollectorresults API
type ImageCollectorResult struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status ImageCollectorResultStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ImageCollectorResultList contains a list of ImageCollectorResult
type ImageCollectorResultList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageCollectorResult `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageCollectorResult{}, &ImageCollectorResultList{})
}
