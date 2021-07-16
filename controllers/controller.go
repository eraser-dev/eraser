package controllers

import (
	"github.com/Azure/eraser/controllers/imagejob"
	"github.com/Azure/eraser/controllers/imagelist"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	controllerLog      = ctrl.Log.WithName("controllerRuntimeLogger")
	controllerAddFuncs []func(manager.Manager) error
)

func init() {
	controllerAddFuncs = append(controllerAddFuncs, imagelist.Add)
	controllerAddFuncs = append(controllerAddFuncs, imagejob.Add)
}

func SetupWithManager(m manager.Manager) error {
	for _, f := range controllerAddFuncs {
		if err := f(m); err != nil {
			if kindMatchErr, ok := err.(*meta.NoKindMatchError); ok {
				controllerLog.Info("CRD %v is not installed, its controller will perform noops!", kindMatchErr.GroupKind)
				continue
			}
			return err
		}
	}
	return nil
}
