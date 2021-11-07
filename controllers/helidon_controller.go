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

package controllers

import (
	"context"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	ochacafev1alpha1 "github.com/oracle-japan/ochacafe-operator-intro/api/v1alpha1"
)

// HelidonReconciler reconciles a Helidon object
type HelidonReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ochacafe.oracle.com,resources=helidons,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ochacafe.oracle.com,resources=helidons/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ochacafe.oracle.com,resources=helidons/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Helidon object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *HelidonReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	// 1. Fetch the Helidon instance
	helidon := &ochacafev1alpha1.Helidon{}
	err := r.Get(ctx, req.NamespacedName, helidon)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("1. Fetch the Helidon instance. Helidon resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, err
		}
		// Error reading the object - requeue the request.
		log.Error(err, "1. Fetch the Helidon instance. Failed to get Helidon")
		return ctrl.Result{}, err
	}
	log.Info("1. Fetch the Helidon instance. Helidon resource found", "helidon.Name", helidon.Name, "helidon.Namespace", helidon.Namespace)

	// 2. Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: helidon.Name, Namespace: helidon.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForHelidon(helidon)
		log.Info("2. Check if the deployment already exists, if not create a new one. Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "2. Check if the deployment already exists, if not create a new one. Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "2. Check if the deployment already exists, if not create a new one. Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// 3. Ensure the deployment size is the same as the spec
	size := helidon.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "3. Ensure the deployment size is the same as the spec. Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		log.Info("3. Ensure the deployment size is the same as the spec. Update deployment size", "Deployment.Spec.Replicas", size)
		return ctrl.Result{Requeue: true}, nil
	}

	// 4. Update the Helidon status with the pod names
	// List the pods for this helidon's deployment
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(helidon.Namespace),
		client.MatchingLabels(labelsForHelidon(helidon.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		log.Error(err, "4. Update the Helidon status with the pod names. Failed to list pods", "Helidon.Namespace", helidon.Namespace, "Helidon.Name", helidon.Name)
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)
	log.Info("4. Update the Helidon status with the pod names. Pod list", "podNames", podNames)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, helidon.Status.Nodes) {
		helidon.Status.Nodes = podNames
		err := r.Status().Update(ctx, helidon)
		if err != nil {
			log.Error(err, "4. Update the Helidon status with the pod names. Failed to update Helidon status")
			return ctrl.Result{}, err
		}
	}
	log.Info("4. Update the Helidon status with the pod names. Update helidon.Status", "helidon.Status.Nodes", helidon.Status.Nodes)

	return ctrl.Result{}, nil
}

// deploymentForHelidon returns a helidon Deployment object
func (r *HelidonReconciler) deploymentForHelidon(m *ochacafev1alpha1.Helidon) *appsv1.Deployment {
	ls := labelsForHelidon(m.Name)
	replicas := m.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "schnatterer/helidon-getting-started",
						Name:  "helidon-app",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8080,
						}},
					}},
				},
			},
		},
	}
	// Set Helidon instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// labelsForHelidon returns the labels for selecting the resources
// belonging to the given helidon CR name.
func labelsForHelidon(name string) map[string]string {
	return map[string]string{"app": "helidon", "helidon_cr": name}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelidonReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ochacafev1alpha1.Helidon{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
