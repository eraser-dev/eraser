package v1alpha3

import (
	unversioned "github.com/eraser-dev/eraser/api/unversioned"
	conversion "k8s.io/apimachinery/pkg/conversion"
)

//nolint:unused
func Convert_v1alpha3_ManagerConfig_To_unversioned_ManagerConfig(in *ManagerConfig, out *unversioned.ManagerConfig, s conversion.Scope) error {
	return autoConvert_v1alpha3_ManagerConfig_To_unversioned_ManagerConfig(in, out, s)
}

// TODO: change this to use unversioned.RuntimeSpec when unversioned is updated
//
//nolint:unused
func autoConvert_v1alpha3_RuntimeSpec_To_unversioned_Runtime(in *RuntimeSpec, out *unversioned.Runtime, _ conversion.Scope) error {
	*out = unversioned.Runtime(in.Name)
	return nil
}

//nolint:unused
func Convert_v1alpha3_RuntimeSpec_To_unversioned_Runtime(in *RuntimeSpec, out *unversioned.Runtime, s conversion.Scope) error {
	return autoConvert_v1alpha3_RuntimeSpec_To_unversioned_Runtime(in, out, s)
}

//nolint:unused
func Convert_unversioned_ManagerConfig_To_v1alpha3_ManagerConfig(in *unversioned.ManagerConfig, out *ManagerConfig, s conversion.Scope) error {
	return autoConvert_unversioned_ManagerConfig_To_v1alpha3_ManagerConfig(in, out, s)
}

// TODO: change this to use unversioned.RuntimeSpec when unversioned is updated
//
//nolint:unused
func autoConvert_unversioned_Runtime_To_v1alpha3_RuntimeSpec(in *unversioned.Runtime, out *RuntimeSpec, _ conversion.Scope) error {
	out.Name = Runtime(string(*in))
	out.Address = RuntimeAddress("")
	return nil
}

//nolint:unused
func Convert_unversioned_Runtime_To_v1alpha3_RuntimeSpec(in *unversioned.Runtime, out *RuntimeSpec, s conversion.Scope) error {
	return autoConvert_unversioned_Runtime_To_v1alpha3_RuntimeSpec(in, out, s)
}
