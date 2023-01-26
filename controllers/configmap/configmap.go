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

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/api/v1alpha1/config"
	controllerUtils "github.com/Azure/eraser/controllers/util"
	"github.com/Azure/eraser/pkg/metrics"
	eraserUtils "github.com/Azure/eraser/pkg/utils"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	configmapName = "eraser-manager-config"
)

var (
	log      = logf.Log.WithName("controller").WithValues("process", "imagelist-controller")
	exporter sdkmetric.Exporter
	reader   sdkmetric.Reader
	provider *sdkmetric.MeterProvider

	configmap = types.NamespacedName{
		Namespace: eraserUtils.GetNamespace(),
		Name:      configmapName,
	}
)

// ImageListReconciler reconciles a ImageList object.
type Reconciler struct {
	client.Client
	scheme       *runtime.Scheme
	eraserConfig eraserv1alpha1.EraserConfig
}

func Add(mgr manager.Manager, cfg *eraserv1alpha1.EraserConfig) error {
	r := newReconciler(mgr, cfg)

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

				log.Info("configmap was updated, rebooting...")
				return true
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				cfg, ok := e.Object.(*corev1.ConfigMap)
				n := types.NamespacedName{Namespace: cfg.GetNamespace(), Name: cfg.GetName()}

				if !ok || n != configmap {
					return false
				}

				log.Info("configmap was deleted, shutting down...")
				return true
			},
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
func newReconciler(mgr manager.Manager, cfg *eraserv1alpha1.EraserConfig) reconcile.Reconciler {
	config := *config.Default()
	if cfg != nil {
		config = *cfg
	}

	otlpEndpoint := config.Manager.OTLPEndpoint
	if otlpEndpoint != "" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		exporter, reader, provider = metrics.ConfigureMetrics(ctx, log, otlpEndpoint)
		global.SetMeterProvider(provider)
	}

	rec := &Reconciler{
		Client:       mgr.GetClient(),
		scheme:       mgr.GetScheme(),
		eraserConfig: config,
	}

	return rec
}

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch,delete
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pods := corev1.PodList{}
	err := r.List(ctx, &pods, client.MatchingLabels{
		"control-plane": "controller-manager",
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	log.V(1).Info("pods", "pods", pods.Items)
	if len(pods.Items) == 0 {
		return ctrl.Result{}, nil
	}

	pod := pods.Items[0]
	if len(pods.Items) > 1 {
		for _, p := range pods.Items[1:] {
			if p.Status.Phase == corev1.PodPhase(corev1.PodRunning) {
				pod = p
				break
			}
		}
	}

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
