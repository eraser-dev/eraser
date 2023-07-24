package controllers

import (
	"errors"

	"github.com/eraser-dev/eraser/api/unversioned/config"
	"github.com/eraser-dev/eraser/controllers/configmap"
	"github.com/eraser-dev/eraser/controllers/imagecollector"
	"github.com/eraser-dev/eraser/controllers/imagejob"
	"github.com/eraser-dev/eraser/controllers/imagelist"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type controllerSetupFunc func(manager.Manager, *config.Manager) error

var (
	controllerLog = ctrl.Log.WithName("controllerRuntimeLogger")

	controllerAddFuncs = []controllerSetupFunc{
		imagelist.Add,
		imagejob.Add,
		imagecollector.Add,
		configmap.Add,
	}
)

func SetupWithManager(m manager.Manager, cfg *config.Manager) error {
	controllerLog.Info("set up with manager")
	for _, f := range controllerAddFuncs {
		if err := f(m, cfg); err != nil {
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
