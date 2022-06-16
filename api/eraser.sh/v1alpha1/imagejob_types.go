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

// ImageJobSpec defines the desired state of ImageJob.
type ImageJobSpec struct {
	// Specifies the job that will be created when executing an ImageJob.
	JobTemplate v1.PodTemplateSpec `json:"template"`
}

// JobPhase defines the phase of an ImageJob status.
type JobPhase string

const (
	PhaseRunning   JobPhase = "Running"
	PhaseCompleted JobPhase = "Completed"
	PhaseFailed    JobPhase = "Failed"
)

// ImageJobStatus defines the observed state of ImageJob.
type ImageJobStatus struct {
	// number of pods that failed
	Failed int `json:"failed"`

	// number of pods that completed successfully
	Succeeded int `json:"succeeded"`

	// desired number of pods
	Desired int `json:"desired"`

	// number of nodes that were skipped e.g. because they are not a linux node
	Skipped int `json:"skipped"`

	// job running, successfully completed, or failed
	Phase JobPhase `json:"phase"`

	// Time to delay deletion until
	DeleteAfter *metav1.Time `json:"deleteAfter,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope="Cluster"
// ImageJob is the Schema for the imagejobs API.
type ImageJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageJobSpec   `json:"spec,omitempty"`
	Status ImageJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ImageJobList contains a list of ImageJob.
type ImageJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageJob `json:"items"`
}

func init() {
	localSchemeBuilder.Register(addKnownTypes)
}
