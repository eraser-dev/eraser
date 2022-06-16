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

type ImageCollectorSpec struct {
	Images []Image `json:"images"`
}

type Image struct {
	Digest string `json:"digest"`
	Name   string `json:"name,omitempty"`
}

type ImageCollectorStatus struct {
	Vulnerable []Image `json:"vulnerable,omitempty"`
	Failed     []Image `json:"failed,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope="Cluster"
// +genclient
type ImageCollector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageCollectorSpec   `json:"spec,omitempty"`
	Status ImageCollectorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type ImageCollectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageCollector `json:"items"`
}

func init() {
	localSchemeBuilder.Register(addKnownTypes)
}
