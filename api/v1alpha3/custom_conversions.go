package v1alpha3

import (
	unversioned "github.com/eraser-dev/eraser/api/unversioned"
	conversion "k8s.io/apimachinery/pkg/conversion"
)

// The following were added manually for ManagerConfig RuntimeSpec conversions:

func Convert_v1alpha3_ManagerConfig_To_unversioned_ManagerConfig(in *ManagerConfig, out *unversioned.ManagerConfig, s conversion.Scope) error {
	return autoConvert_v1alpha3_ManagerConfig_To_unversioned_ManagerConfig(in, out, s)
}

// TODO: change this to use unversioned.RuntimeSpec when unversioned is updated
func autoConvert_v1alpha3_RuntimeSpec_To_unversioned_Runtime(in *RuntimeSpec, out *unversioned.Runtime, s conversion.Scope) error {
	*out = unversioned.Runtime(in.Name)
	return nil
}

func Convert_v1alpha3_RuntimeSpec_To_unversioned_Runtime(in *RuntimeSpec, out *unversioned.Runtime, s conversion.Scope) error {
	return autoConvert_v1alpha3_RuntimeSpec_To_unversioned_Runtime(in, out, s)
}

func Convert_unversioned_ManagerConfig_To_v1alpha3_ManagerConfig(in *unversioned.ManagerConfig, out *ManagerConfig, s conversion.Scope) error {
	return autoConvert_unversioned_ManagerConfig_To_v1alpha3_ManagerConfig(in, out, s)
}

// TODO: change this to use unversioned.RuntimeSpec when unversioned is updated
func autoConvert_unversioned_Runtime_To_v1alpha3_RuntimeSpec(in *unversioned.Runtime, out *RuntimeSpec, s conversion.Scope) error {
	out.Name = Runtime(string(*in))
	out.Address = RuntimeAddress("")
	return nil
}

func Convert_unversioned_Runtime_To_v1alpha3_RuntimeSpec(in *unversioned.Runtime, out *RuntimeSpec, s conversion.Scope) error {
	return autoConvert_unversioned_Runtime_To_v1alpha3_RuntimeSpec(in, out, s)
}
