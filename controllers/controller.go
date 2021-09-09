package controllers

import (
	"errors"

	"github.com/Azure/eraser/controllers/imagejob"
	"github.com/Azure/eraser/controllers/imagelist"
	"github.com/Azure/eraser/controllers/options"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	controllerLog      = ctrl.Log.WithName("controllerRuntimeLogger")
	controllerAddFuncs []func(manager.Manager, options.Options) error
)

func init() {
	controllerAddFuncs = append(controllerAddFuncs, imagelist.Add)
	controllerAddFuncs = append(controllerAddFuncs, imagejob.Add)
}

func SetupWithManager(m manager.Manager, eraserImage string) error {
	opt := options.Options{EraserImage: eraserImage}
	for _, f := range controllerAddFuncs {
		if err := f(m, opt); err != nil {
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
