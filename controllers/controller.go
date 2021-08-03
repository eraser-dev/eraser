package controllers

import (
<<<<<<< HEAD
	"errors"

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
=======
	"github.com/Azure/eraser/controllers/imagejob"
	"github.com/Azure/eraser/controllers/imagelist"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var controllerAddFuncs []func(manager.Manager) error
>>>>>>> 6bd9875f650ff349c33bd2b85d8eb5e786ce4a58

func init() {
	controllerAddFuncs = append(controllerAddFuncs, imagelist.Add)
	controllerAddFuncs = append(controllerAddFuncs, imagejob.Add)
}

func SetupWithManager(m manager.Manager) error {
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
