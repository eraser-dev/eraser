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

package unversioned

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	Duration       time.Duration
	Runtime        string
	RuntimeAddress string

	RuntimeSpec struct {
		Name    Runtime `json:"name"`
		Address string  `json:"address"`
	}
)

const (
	RuntimeContainerd Runtime = "containerd"
	RuntimeDockerShim Runtime = "dockershim"
	RuntimeCrio       Runtime = "crio"

	DockerPath     = "/run/dockershim.sock"
	ContainerdPath = "/run/containerd/containerd.sock"
	CrioPath       = "/run/crio/crio.sock"
)

// Does not set the env vars, but returns the key and value so that the caller
// can set them.
func (r *RuntimeSpec) GetSocketEnvVars() (string, string, error) {
	switch r.Name {
	case RuntimeContainerd:
		return "CONTAINERD_ADDRESS", r.Address, nil
	case RuntimeDockerShim:
		return "DOCKER_HOST", r.Address, nil
	case RuntimeCrio:
		// For trivy, the socket MUST be at $XDG_RUNTIME_DIR/podman/podman.sock.
		// Currently, there is no way for trivy to recognize the podman socket
		// at another address, although $XDG_RUNTIME_DIR may be modified.
		baseDir := filepath.Dir(r.Address)
		runtimeDir := filepath.Dir(baseDir)
		return "XDG_RUNTIME_DIR", runtimeDir, nil
	}

	return "", "", fmt.Errorf("invalid runtime: valid names are %s, %s, %s", RuntimeContainerd, RuntimeDockerShim, RuntimeCrio)
}

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

func ConvertRuntimeToRuntimeAddress(r Runtime) (RuntimeAddress, error) {
	var rr RuntimeAddress
	switch rt := Runtime(r); rt {
	case RuntimeContainerd:
		rr = RuntimeAddress(fmt.Sprintf("unix://%s", ContainerdPath))
	case RuntimeDockerShim:
		rr = RuntimeAddress(fmt.Sprintf("unix://%s", DockerPath))
	case RuntimeCrio:
		rr = RuntimeAddress(fmt.Sprintf("unix://%s", CrioPath))
	default:
		u, err := url.Parse(string(r))
		if err != nil {
			return RuntimeAddress(""), err
		}

		switch u.Scheme {
		case "tcp", "unix":
		default:
			return RuntimeAddress(""), fmt.Errorf("invalid scheme: valid schemes for runtime socket address are `tcp` and `unix`")
		}

		rr = RuntimeAddress(r)
	}

	return rr, nil
}

func ConvertRuntimeToRuntimeSpec(r Runtime) (RuntimeSpec, error) {
	var rr RuntimeSpec

	switch rt := Runtime(r); rt {
	case RuntimeContainerd:
		rr = RuntimeSpec{Name: RuntimeContainerd, Address: fmt.Sprintf("unix://%s", ContainerdPath)}
	case RuntimeDockerShim:
		rr = RuntimeSpec{Name: RuntimeDockerShim, Address: fmt.Sprintf("unix://%s", DockerPath)}
	case RuntimeCrio:
		rr = RuntimeSpec{Name: RuntimeCrio, Address: fmt.Sprintf("unix://%s", CrioPath)}
	default:
		return rr, fmt.Errorf("invalid runtime: valid names are %s, %s, %s", RuntimeContainerd, RuntimeDockerShim, RuntimeCrio)
	}

	return rr, nil
}

func (r *RuntimeAddress) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}

	switch rt := Runtime(str); rt {
	case RuntimeContainerd:
		*r = RuntimeAddress(fmt.Sprintf("unix://%s", ContainerdPath))
	case RuntimeDockerShim:
		*r = RuntimeAddress(fmt.Sprintf("unix://%s", DockerPath))
	case RuntimeCrio:
		*r = RuntimeAddress(fmt.Sprintf("unix://%s", CrioPath))
	default:
		u, err := url.Parse(str)
		if err != nil {
			return err
		}

		switch u.Scheme {
		case "tcp", "unix":
		default:
			return fmt.Errorf("invalid scheme: valid schemes for runtime socket address are `tcp` and `unix`")
		}

		*r = RuntimeAddress(str)
	}

	return nil
}

func (r *RuntimeSpec) UnmarshalJSON(b []byte) error {
	var rr RuntimeSpec
	err := json.Unmarshal(b, &rr)
	if err != nil {
		return err
	}

	switch rt := rr.Name; rt {
	case RuntimeContainerd, RuntimeDockerShim, RuntimeCrio:
		if rr.Address != "" {
			*r = rr
			return nil
		}

		converted, err := ConvertRuntimeToRuntimeSpec(rt)
		if err != nil {
			return err
		}

		*r = converted
	default:
		return fmt.Errorf("invalid runtime: valid names are %s, %s, %s", RuntimeContainerd, RuntimeDockerShim, RuntimeCrio)
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
	Runtime           RuntimeSpec      `json:"runtime,omitempty"`
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
