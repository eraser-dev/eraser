//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha2

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Components) DeepCopyInto(out *Components) {
	*out = *in
	in.Collector.DeepCopyInto(&out.Collector)
	in.Scanner.DeepCopyInto(&out.Scanner)
	in.Remover.DeepCopyInto(&out.Remover)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Components.
func (in *Components) DeepCopy() *Components {
	if in == nil {
		return nil
	}
	out := new(Components)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerConfig) DeepCopyInto(out *ContainerConfig) {
	*out = *in
	out.Image = in.Image
	in.Request.DeepCopyInto(&out.Request)
	in.Limit.DeepCopyInto(&out.Limit)
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerConfig.
func (in *ContainerConfig) DeepCopy() *ContainerConfig {
	if in == nil {
		return nil
	}
	out := new(ContainerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EraserConfig) DeepCopyInto(out *EraserConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.Manager.DeepCopyInto(&out.Manager)
	in.Components.DeepCopyInto(&out.Components)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EraserConfig.
func (in *EraserConfig) DeepCopy() *EraserConfig {
	if in == nil {
		return nil
	}
	out := new(EraserConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *EraserConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Image) DeepCopyInto(out *Image) {
	*out = *in
	if in.Names != nil {
		in, out := &in.Names, &out.Names
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Digests != nil {
		in, out := &in.Digests, &out.Digests
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Image.
func (in *Image) DeepCopy() *Image {
	if in == nil {
		return nil
	}
	out := new(Image)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageJob) DeepCopyInto(out *ImageJob) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageJob.
func (in *ImageJob) DeepCopy() *ImageJob {
	if in == nil {
		return nil
	}
	out := new(ImageJob)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ImageJob) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageJobCleanupConfig) DeepCopyInto(out *ImageJobCleanupConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageJobCleanupConfig.
func (in *ImageJobCleanupConfig) DeepCopy() *ImageJobCleanupConfig {
	if in == nil {
		return nil
	}
	out := new(ImageJobCleanupConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageJobConfig) DeepCopyInto(out *ImageJobConfig) {
	*out = *in
	out.Cleanup = in.Cleanup
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageJobConfig.
func (in *ImageJobConfig) DeepCopy() *ImageJobConfig {
	if in == nil {
		return nil
	}
	out := new(ImageJobConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageJobList) DeepCopyInto(out *ImageJobList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ImageJob, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageJobList.
func (in *ImageJobList) DeepCopy() *ImageJobList {
	if in == nil {
		return nil
	}
	out := new(ImageJobList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ImageJobList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageJobStatus) DeepCopyInto(out *ImageJobStatus) {
	*out = *in
	if in.DeleteAfter != nil {
		in, out := &in.DeleteAfter, &out.DeleteAfter
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageJobStatus.
func (in *ImageJobStatus) DeepCopy() *ImageJobStatus {
	if in == nil {
		return nil
	}
	out := new(ImageJobStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageList) DeepCopyInto(out *ImageList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageList.
func (in *ImageList) DeepCopy() *ImageList {
	if in == nil {
		return nil
	}
	out := new(ImageList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ImageList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageListList) DeepCopyInto(out *ImageListList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ImageList, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageListList.
func (in *ImageListList) DeepCopy() *ImageListList {
	if in == nil {
		return nil
	}
	out := new(ImageListList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ImageListList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageListSpec) DeepCopyInto(out *ImageListSpec) {
	*out = *in
	if in.Images != nil {
		in, out := &in.Images, &out.Images
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageListSpec.
func (in *ImageListSpec) DeepCopy() *ImageListSpec {
	if in == nil {
		return nil
	}
	out := new(ImageListSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageListStatus) DeepCopyInto(out *ImageListStatus) {
	*out = *in
	if in.Timestamp != nil {
		in, out := &in.Timestamp, &out.Timestamp
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageListStatus.
func (in *ImageListStatus) DeepCopy() *ImageListStatus {
	if in == nil {
		return nil
	}
	out := new(ImageListStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ManagerConfig) DeepCopyInto(out *ManagerConfig) {
	*out = *in
	out.Scheduling = in.Scheduling
	out.Profile = in.Profile
	out.ImageJob = in.ImageJob
	if in.PullSecrets != nil {
		in, out := &in.PullSecrets, &out.PullSecrets
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.NodeFilter.DeepCopyInto(&out.NodeFilter)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ManagerConfig.
func (in *ManagerConfig) DeepCopy() *ManagerConfig {
	if in == nil {
		return nil
	}
	out := new(ManagerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeFilterConfig) DeepCopyInto(out *NodeFilterConfig) {
	*out = *in
	if in.Selectors != nil {
		in, out := &in.Selectors, &out.Selectors
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeFilterConfig.
func (in *NodeFilterConfig) DeepCopy() *NodeFilterConfig {
	if in == nil {
		return nil
	}
	out := new(NodeFilterConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OptionalContainerConfig) DeepCopyInto(out *OptionalContainerConfig) {
	*out = *in
	in.ContainerConfig.DeepCopyInto(&out.ContainerConfig)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OptionalContainerConfig.
func (in *OptionalContainerConfig) DeepCopy() *OptionalContainerConfig {
	if in == nil {
		return nil
	}
	out := new(OptionalContainerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProfileConfig) DeepCopyInto(out *ProfileConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProfileConfig.
func (in *ProfileConfig) DeepCopy() *ProfileConfig {
	if in == nil {
		return nil
	}
	out := new(ProfileConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RepoTag) DeepCopyInto(out *RepoTag) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepoTag.
func (in *RepoTag) DeepCopy() *RepoTag {
	if in == nil {
		return nil
	}
	out := new(RepoTag)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceRequirements) DeepCopyInto(out *ResourceRequirements) {
	*out = *in
	out.Mem = in.Mem.DeepCopy()
	out.CPU = in.CPU.DeepCopy()
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceRequirements.
func (in *ResourceRequirements) DeepCopy() *ResourceRequirements {
	if in == nil {
		return nil
	}
	out := new(ResourceRequirements)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScheduleConfig) DeepCopyInto(out *ScheduleConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScheduleConfig.
func (in *ScheduleConfig) DeepCopy() *ScheduleConfig {
	if in == nil {
		return nil
	}
	out := new(ScheduleConfig)
	in.DeepCopyInto(out)
	return out
}
