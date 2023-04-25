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

package v1alpha2

import (
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	Duration time.Duration
	Runtime  string
)

const (
	RuntimeContainerd Runtime = "containerd"
	RuntimeDockerShim Runtime = "dockershim"
	RuntimeCrio       Runtime = "crio"
)

func (td *Duration) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}

	pd, err := time.ParseDuration(str)
	if err != nil {
		return err
	}

	*td = Duration(pd)
	return nil
}

func (r *Runtime) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}

	switch rt := Runtime(str); rt {
	case RuntimeContainerd, RuntimeDockerShim, RuntimeCrio:
		*r = rt
	default:
		return fmt.Errorf("cannot determine runtime type: %s. valid values are containerd, dockershim, or crio", str)
	}

	return nil
}

func (td *Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, time.Duration(*td).String())), nil
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type OptionalContainerConfig struct {
	Enabled         bool `json:"enabled,omitempty"`
	ContainerConfig `json:",inline"`
}

type ContainerConfig struct {
	Image   RepoTag              `json:"image,omitempty"`
	Request ResourceRequirements `json:"request,omitempty"`
	Limit   ResourceRequirements `json:"limit,omitempty"`
	Config  *string              `json:"config,omitempty"`
}

type ManagerConfig struct {
	Runtime           Runtime          `json:"runtime,omitempty"`
	OTLPEndpoint      string           `json:"otlpEndpoint,omitempty"`
	LogLevel          string           `json:"logLevel,omitempty"`
	Scheduling        ScheduleConfig   `json:"scheduling,omitempty"`
	Profile           ProfileConfig    `json:"profile,omitempty"`
	ImageJob          ImageJobConfig   `json:"imageJob,omitempty"`
	PullSecrets       []string         `json:"pullSecrets,omitempty"`
	NodeFilter        NodeFilterConfig `json:"nodeFilter,omitempty"`
	PriorityClassName string           `json:"priorityClassName,omitempty"`
}

type ScheduleConfig struct {
	RepeatInterval   Duration `json:"repeatInterval,omitempty"`
	BeginImmediately bool     `json:"beginImmediately,omitempty"`
}

type ProfileConfig struct {
	Enabled bool `json:"enabled,omitempty"`
	Port    int  `json:"port,omitempty"`
}

type ImageJobConfig struct {
	SuccessRatio float64               `json:"successRatio,omitempty"`
	Cleanup      ImageJobCleanupConfig `json:"cleanup,omitempty"`
}

type ImageJobCleanupConfig struct {
	DelayOnSuccess Duration `json:"delayOnSuccess,omitempty"`
	DelayOnFailure Duration `json:"delayOnFailure,omitempty"`
}

type NodeFilterConfig struct {
	Type      string   `json:"type,omitempty"`
	Selectors []string `json:"selectors,omitempty"`
}

type ResourceRequirements struct {
	Mem resource.Quantity `json:"mem,omitempty"`
	CPU resource.Quantity `json:"cpu,omitempty"`
}

type RepoTag struct {
	Repo string `json:"repo,omitempty"`
	Tag  string `json:"tag,omitempty"`
}

type Components struct {
	Collector OptionalContainerConfig `json:"collector,omitempty"`
	Scanner   OptionalContainerConfig `json:"scanner,omitempty"`
	Remover   ContainerConfig         `json:"remover,omitempty"`
}

//+kubebuilder:object:root=true

// EraserConfig is the Schema for the eraserconfigs API.
type EraserConfig struct {
	metav1.TypeMeta `json:",inline"`
	Manager         ManagerConfig `json:"manager"`
	Components      Components    `json:"components"`
}

func init() {
	SchemeBuilder.Register(&EraserConfig{})
}
