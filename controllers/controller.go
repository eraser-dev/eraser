package controllers

import (
	"errors"

	"github.com/Azure/eraser/controllers/imagecollector"
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
	controllerAddFuncs = append(controllerAddFuncs, imagelist.Add, imagejob.Add, imagecollector.Add)
}

func SetupWithManager(m manager.Manager) error {
	controllerLog.Info("set up with manager")
	for _, f := range controllerAddFuncs {
		if err := f(m); err != nil {
			var kindMatchErr *meta.NoKindMatchError
			if errors.As(err, &kindMatchErr) {
				controllerLog.Info("CRD %v is not installed", kindMatchErr.GroupKind)
				continue
			}
			return err
		}
	}
	return nil
}
