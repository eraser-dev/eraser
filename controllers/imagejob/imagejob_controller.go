/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package imagejob

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	controllerLog = ctrl.Log.WithName("controllerRuntimeLogger")
)

func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &Reconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

// ImageJobReconciler reconciles a ImageJob object
type Reconciler struct {
	client.Client
	scheme *runtime.Scheme
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("imagejob-controller", mgr, controller.Options{
		Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to ImageJob
	err = c.Watch(&source.Kind{Type: &eraserv1alpha1.ImageJob{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to pods created by ImageJob (eraser pods)
	err = c.Watch(&source.Kind{Type: &v1.Pod{}}, &handler.EnqueueRequestForOwner{OwnerType: &eraserv1alpha1.ImageJob{}})

	return nil
}

//+kubebuilder:rbac:groups=eraser.sh,resources=imagejobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=eraser.sh,resources=imagejobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=eraser.sh,resources=imagejobs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update;create;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ImageJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	controllerLog.Info("imagejob reconcile")

	job := &eraserv1alpha1.ImageJob{}
	err := r.Get(context.TODO(), req.NamespacedName, job)
	if err != nil {
		panic(err)
	}

	if isJobComplete(job) {
		// update imagejobstatus, phase completed, update message
		// update imagestatus's and update imagelist

	} else {
		nodes := &v1.NodeList{}
		err = r.List(context.TODO(), nodes)
		if err != nil {
			panic(err)
		}

		count := 0

		// only get names from for loop
		for _, n := range nodes.Items {
			count++
			controllerLog.Info("inside nodes.Items for loop")
			nodeName := n.Name

			runTime := n.Status.NodeInfo.ContainerRuntimeVersion
			runTimeName := strings.Split(runTime, ":")[0]

			var socketPath string

			if runTimeName == "dockershim" {
				socketPath = "/var/run/dockershim.sock"
			} else if runTimeName == "containerd" {
				socketPath = "/run/containerd/containerd.sock"
			} else if runTimeName == "crio" {
				socketPath = "/var/run/crio/crio.sock "
			} else {
				log.Println("runtime not compatible")
				os.Exit(1)
			}

			imageJob := &eraserv1alpha1.ImageJob{}

			err := r.Get(ctx, req.NamespacedName, imageJob)
			if err != nil {
				controllerLog.Info("err")
				panic(err)
			}
			image := imageJob.Spec.JobTemplate.Spec.Containers[0]
			image.Args = append(image.Args, "--runtime="+runTimeName)
			image.VolumeMounts = []v1.VolumeMount{{MountPath: socketPath, Name: runTimeName + "-sock-volume"}}

			podSpec := imageJob.Spec.JobTemplate.Spec
			podSpec.Volumes = []v1.Volume{{Name: runTimeName + "-sock-volume", VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: socketPath}}}}
			podName := image.Name + strconv.Itoa(count)
			podSpec.NodeName = nodeName
			podSpec.Containers = []v1.Container{image}

			pod := &v1.Pod{
				TypeMeta:   metav1.TypeMeta{},
				Spec:       podSpec,
				ObjectMeta: metav1.ObjectMeta{Namespace: "eraser-system", Name: podName, Labels: map[string]string{"name": image.Name}},
			}

			// TODO: check if pod fits and can be scheduled on node
			err = r.Create(context.TODO(), pod)
			if err != nil {
				controllerLog.Info("err")
				panic(err)
			}
			controllerLog.Info("created pod")
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&eraserv1alpha1.ImageJob{}).
		Complete(r)
}

func podsFinished(pods v1.PodList) bool {
	for _, p := range pods.Items {
		if IsPodActive(&p) {
			return false
		}
	}
	return true
}

// couldnt import kubecontroller "k8s.io/kubernetes/pkg/controller"
func IsPodActive(p *v1.Pod) bool {
	return v1.PodSucceeded != p.Status.Phase &&
		v1.PodFailed != p.Status.Phase &&
		p.DeletionTimestamp == nil
}

func isJobComplete(job *eraserv1alpha1.ImageJob)
