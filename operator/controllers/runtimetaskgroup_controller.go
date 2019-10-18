/*
Copyright 2019 The Kubernetes Authors.

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

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "k8s.io/kubeadm/operator/api/v1alpha1"
)

// RuntimeTaskGroupReconciler reconciles a RuntimeTaskGroup object
type RuntimeTaskGroupReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=operator.kubeadm.x-k8s.io,resources=runtimetaskgroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kubeadm.x-k8s.io,resources=runtimetaskgroups/status,verbs=get;update;patch

// Reconcile a runtimetaskgroup
func (r *RuntimeTaskGroupReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("runtimetaskgroup", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager configures the controller for calling the reconciler
func (r *RuntimeTaskGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.RuntimeTaskGroup{}).
		Complete(r)
}
