package configmap

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"go.opentelemetry.io/otel/metric/global"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eraser-dev/eraser/api/unversioned/config"
	eraserv1 "github.com/eraser-dev/eraser/api/v1"
	controllerUtils "github.com/eraser-dev/eraser/controllers/util"
	"github.com/eraser-dev/eraser/pkg/metrics"
	eraserUtils "github.com/eraser-dev/eraser/pkg/utils"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	log      = logf.Log.WithName("controller").WithValues("process", "configmap-controller")
	provider *sdkmetric.MeterProvider

	configmap = types.NamespacedName{
		Namespace: eraserUtils.GetNamespace(),
		Name:      controllerUtils.EraserConfigmapName,
	}
)

// ImageListReconciler reconciles a ImageList object.
type Reconciler struct {
	client.Client
	scheme       *runtime.Scheme
	eraserConfig *config.Manager
}

func Add(mgr manager.Manager, cfg *config.Manager) error {
	r, err := newReconciler(mgr, cfg)
	if err != nil {
		return err
	}

	c, err := controller.New("imagelist-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return err
	}

	err = c.Watch(
		&source.Kind{Type: &corev1.ConfigMap{}},
		&handler.EnqueueRequestForObject{},
		predicate.ResourceVersionChangedPredicate{},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				cfg, ok := e.ObjectNew.(*corev1.ConfigMap)
				n := types.NamespacedName{Namespace: cfg.GetNamespace(), Name: cfg.GetName()}

				if !ok || n != configmap {
					return false
				}

				log.Info("configmap was updated, reloading")
				return true
			},
			DeleteFunc:  controllerUtils.NeverOnDelete,
			GenericFunc: controllerUtils.NeverOnGeneric,
			CreateFunc:  controllerUtils.NeverOnCreate,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager, cfg *config.Manager) (reconcile.Reconciler, error) {
	c, err := cfg.Read()
	if err != nil {
		return nil, err
	}

	otlpEndpoint := c.Manager.OTLPEndpoint
	if otlpEndpoint != "" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		_, _, provider = metrics.ConfigureMetrics(ctx, log, otlpEndpoint)
		global.SetMeterProvider(provider)
	}

	rec := &Reconciler{
		Client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		eraserConfig: cfg,
	}

	return rec, nil
}

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch,delete
func (r *Reconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	j := eraserv1.ImageJobList{}
	err := r.List(ctx, &j)
	if err != nil {
		return ctrl.Result{}, err
	}

	jobs := j.Items
	for i := range jobs {
		if jobs[i].Status.Phase == eraserv1.PhaseRunning {
			return ctrl.Result{}, fmt.Errorf("job is currently running, deferring configmap update")
		}
	}

	p := corev1.PodList{}
	err = r.List(ctx, &p, client.MatchingLabels{
		"control-plane": "controller-manager",
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	pods := p.Items

	if len(pods) == 0 {
		return ctrl.Result{}, nil
	}

	pod := pods[0]
	for i := range pods[1:] {
		if pods[i].Status.Phase == corev1.PodPhase(corev1.PodRunning) {
			pod = pods[i]
			break
		}
	}

	// the configmap is mounted to the filesystem, but the normal
	// reconciliation loop will not update it on the node's filesystem until
	// about 60-90 seconds later. updating the annotations will trigger an
	// almost immediate update, which is monitored by an inotify watch set up in
	// the main() function.
	//
	// the annotation only needs to be different from the previous value, so we
	// don't need cryptographically sound random numbers here. the following
	// comment disables the linter which prefers random numbers from the
	// crypto/rand library.
	//nolint:all
	newVersion := fmt.Sprintf("%d", rand.Int63())
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations["eraser.sh/configVersion"] = newVersion

	err = r.Update(ctx, &pod)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
