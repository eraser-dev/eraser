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

package v1

import (
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ContainerConfig struct {
	// REQUIRED
	Enable  bool                 `json:"enable"`
	Image   RepoTag              `json:"image"`
	Request ResourceRequirements `json:"request"`
	Limit   ResourceRequirements `json:"limit"`
	Config  *string              `json:"config,omitempty"`
}

type ManagerConfig struct {
	Runtime     string           `json:"runtime"`
	LogLevel    string           `json"logLevel"`
	Scheduling  ScheduleConfig   `json:"scheduling"`
	Profile     ProfileConfig    `json:"profile"`
	ImageJob    ImageJobConfig   `json:"imageJob"`
	PullSecrets []string         `json:"pullSecrets"`
	NodeFilter  NodeFilterConfig `json:"nodeFilter"`
}

type ScheduleConfig struct {
	RepeatInterval   time.Duration `json:"repeatInterval"`
	BeginImmediately bool          `json:"beginImmediately"`
}

type ProfileConfig struct {
	Enable bool `json:"enable"`
	Port   int  `json:"port"`
}

type ImageJobConfig struct {
	SuccessRatio float64               `json:"successRatio"`
	Cleanup      ImageJobCleanupConfig `json:"cleanup"`
}

type ImageJobCleanupConfig struct {
	DelayOnSuccess time.Duration `json:"delayOnSuccess"`
	DelayOnFailure time.Duration `json:"delayOnFailure"`
}

type NodeFilterConfig struct {
	Type      string   `json:"type"`
	Selectors []string `json:"selectors"`
}

type ResourceRequirements struct {
	Mem resource.Quantity `json:"mem"`
	CPU resource.Quantity `json:"cpu"`
}

type RepoTag struct {
	Repo string `json:"repo"`
	Tag  string `json:"tag"`
}

type Components struct {
	Collector ContainerConfig `json:"collector"`
	Scanner   ContainerConfig `json:"scanner"`
	Eraser    ContainerConfig `json:"eraser"`
}

//+kubebuilder:object:root=true

// EraserSystemConfig is the Schema for the eraserconfigs API
type EraserSystemConfig struct {
	metav1.TypeMeta `json:",inline"`

	// ControllerManagerConfigurationSpec returns the configurations for controllers
	cfg.ControllerManagerConfigurationSpec `json:",inline"`
	ManagerConfig                          `json:",inline"`

	Components Components `json:"components"`
}

func init() {
	SchemeBuilder.Register(&EraserSystemConfig{})
}
