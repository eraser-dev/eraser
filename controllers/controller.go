package controllers

import (
<<<<<<< HEAD
	"github.com/Azure/eraser/controllers/imagejob"
	"github.com/Azure/eraser/controllers/imagelist"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var controllerAddFuncs []func(manager.Manager) error
=======
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
>>>>>>> main/main

func init() {
	controllerAddFuncs = append(controllerAddFuncs, imagelist.Add)
	controllerAddFuncs = append(controllerAddFuncs, imagejob.Add)
}

func SetupWithManager(m manager.Manager) error {
	for _, f := range controllerAddFuncs {
		if err := f(m); err != nil {
<<<<<<< HEAD
			if kindMatchErr, ok := err.(*meta.NoKindMatchError); ok {
				klog.Infof("CRD %v is not installed, its controller will perform noops!", kindMatchErr.GroupKind)
=======
			var kindMatchErr *meta.NoKindMatchError
			if errors.As(err, &kindMatchErr) {
				controllerLog.Info("CRD %v is not installed", kindMatchErr.GroupKind)
>>>>>>> main/main
				continue
			}
			return err
		}
	}
	return nil
}
