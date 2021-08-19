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

type NodeCleanUpDetail struct {
	ImageName string `json:"imageName"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

// ImageStatusStatus defines the observed state of ImageStatus
type NodeCleanUpResult struct {
	// Specifies on which node image removal took place
	Node string `json:"node"`

	// List of NodeCleanUpDetail that specify image name, status of removal, and message
	Results []NodeCleanUpDetail `json:"results"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope="Cluster"
// ImageStatus is the Schema for the imagestatus API
type ImageStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Result NodeCleanUpResult `json:"result,omitempty"`
}

//+kubebuilder:object:root=true

// ImageStatusList contains a list of NodeCleanUpResults
type ImageStatusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeCleanUpResult `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImageStatus{}, &ImageStatus{})
}
