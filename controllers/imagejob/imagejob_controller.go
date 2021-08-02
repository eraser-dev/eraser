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

const (
	dockerPath     = "/run/dockershim.sock"
	containerdPath = "/run/containerd/containerd.sock"
	crioPath       = "/run/crio/crio.sock"
	docker         = "docker"
	containerd     = "containerd"
	crio           = "cri-o"
)

var (
	controllerLog = ctrl.Log.WithName("imagejob")
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
	nodes := &v1.NodeList{}
	err := r.List(ctx, nodes)
	if err != nil {
		return ctrl.Result{}, err
	}

	// map of node names and runtime
	nodeMap := processNodes(nodes.Items)

	for nodeName, runtime := range nodeMap {
		runtimeName := strings.Split(runtime, ":")[0]
		mountPath := getMountPath(runtimeName)
		if mountPath == "" {
			log.Println("Incompatible runtime on node ", nodeName)
			continue
		}

		imageJob := &eraserv1alpha1.ImageJob{}
		err := r.Get(ctx, req.NamespacedName, imageJob)
		if err != nil {
			controllerLog.Info("No ImageJob: ", req.NamespacedName)
			return ctrl.Result{}, err
		}

		givenImage := imageJob.Spec.JobTemplate.Spec.Containers[0]
		image := v1.Container{
			Args:            append(givenImage.Args, "--runtime="+runtimeName),
			VolumeMounts:    []v1.VolumeMount{{MountPath: mountPath, Name: runtimeName + "-sock-volume"}},
			Image:           givenImage.Image,
			Name:            givenImage.Name,
			ImagePullPolicy: givenImage.ImagePullPolicy,
		}

		givenPodSpec := imageJob.Spec.JobTemplate.Spec
		podSpec := v1.PodSpec{
			RestartPolicy:      givenPodSpec.RestartPolicy,
			ServiceAccountName: givenPodSpec.ServiceAccountName,
			Containers:         []v1.Container{image},
			NodeName:           nodeName,
			Volumes:            []v1.Volume{{Name: runtimeName + "-sock-volume", VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: mountPath}}}},
		}

		podName := image.Name + nodeName
		pod := &v1.Pod{
			TypeMeta:   metav1.TypeMeta{},
			Spec:       podSpec,
			ObjectMeta: metav1.ObjectMeta{Namespace: "eraser-system", Name: podName, Labels: map[string]string{"name": image.Name}},
		}

		// TODO: check if pod fits and can be scheduled on node
		err = r.Create(ctx, pod)
		if err != nil {
			return ctrl.Result{}, err
		}
		controllerLog.Info("created pod", "name", podName, "node", nodeName, "podType", image.Name)

	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&eraserv1alpha1.ImageJob{}).
		Complete(r)
}

func processNodes(nodes []v1.Node) map[string]string {
	m := make(map[string]string, len(nodes))
	for _, n := range nodes {
		m[n.Name] = n.Status.NodeInfo.ContainerRuntimeVersion
	}
	return m
}

func getMountPath(runtimeName string) string {
	switch runtimeName {
	case docker:
		return dockerPath
	case containerd:
		return containerdPath
	case crio:
		return crioPath
	default:
		return ""
	}
}
