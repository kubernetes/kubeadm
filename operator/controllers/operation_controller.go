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
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
	operatorerrors "k8s.io/kubeadm/operator/errors"
	"k8s.io/kubeadm/operator/operations"
)

// OperationReconciler reconciles a Operation object
type OperationReconciler struct {
	client.Client
	ManagerContainerName string
	ManagerNamespace     string
	AgentImage           string
	MetricsRBAC          bool
	Log                  logr.Logger
	recorder             record.EventRecorder
}

// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kubeadm.x-k8s.io,resources=operations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kubeadm.x-k8s.io,resources=operations/status,verbs=get;update;patch

// SetupWithManager configures the controller for calling the reconciler
func (r *OperationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1.Operation{}).
		Owns(&operatorv1.RuntimeTaskGroup{}). // force reconcile operation every time one of the owned TaskGroups change
		Complete(r)

	//TODO: watch DS for operation Daemonsets

	r.recorder = mgr.GetEventRecorderFor("operation-controller")
	return err
}

// Reconcile an operation
func (r *OperationReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, rerr error) {
	ctx := context.Background()
	log := r.Log.WithValues("operation", req.NamespacedName)

	// Fetch the Operation instance
	operation := &operatorv1.Operation{}
	if err := r.Client.Get(ctx, req.NamespacedName, operation); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Ignore the Operation if it is already completed or failed
	if operation.Status.CompletionTime != nil {
		// Reconcile the daemon set that deploys controller agents on nodes, so we are sure it is deleted after completion
		err := r.reconcileDaemonSet(operation, log)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Initialize the patch helper

	patchHelper, err := patch.NewHelper(operation, r)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Always attempt to Patch the Operation object and status after each reconciliation.
	defer func() {
		if err := patchHelper.Patch(ctx, operation); err != nil {
			log.Error(err, "failed to patch Operation")
			if rerr == nil {
				rerr = err
			}
		}
	}()

	// Reconcile the Operation
	if err := r.reconcileOperation(operation, log); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *OperationReconciler) reconcileDaemonSet(operation *operatorv1.Operation, log logr.Logger) error {
	daemonSet, err := getDaemonSet(r.Client, operation, r.ManagerNamespace)
	if err != nil {
		return err
	}

	if daemonSet != nil {
		// if operation completed
		if daemonSetShouldBeRunning(operation) {
			return nil
		}

		log.WithValues("daemonset-name", daemonSetName(operation.Name)).Info("deleting DaemonSet")
		if err := deleteDaemonSet(r.Client, daemonSet); err != nil {
			return err
		}

		return nil
	}

	if !daemonSetShouldBeRunning(operation) {
		return nil
	}

	// if operation running
	log.WithValues("daemonset-name", daemonSetName(operation.Name)).Info("creating DaemonSet")
	image := r.AgentImage
	if image == "" {
		image, err = getImage(r.Client, r.ManagerNamespace, r.ManagerContainerName)
		if err != nil {
			return err
		}
	}

	if err := createDaemonSet(r.Client, operation, r.ManagerNamespace, image, r.MetricsRBAC); err != nil {
		return err
	}

	return nil
}

func daemonSetShouldBeRunning(operation *operatorv1.Operation) bool {
	return operation.Status.CompletionTime == nil &&
		operation.Status.ErrorReason == nil &&
		operation.Status.ErrorMessage == nil
}

func (r *OperationReconciler) reconcileOperation(operation *operatorv1.Operation, log logr.Logger) (err error) {
	// Reconcile paused settings
	r.reconcilePause(operation)

	// Reconcile labels so the operation and the operation object can be searched by a well-known set of labels
	r.reconcileLabels(operation)

	// Reconcile the daemon set that deploys controller agents on nodes
	err = r.reconcileDaemonSet(operation, log)
	if err != nil {
		return
	}

	// Handle deleted Operation
	if !operation.DeletionTimestamp.IsZero() {
		err = r.reconcileDelete(operation)
		if err != nil {
			return
		}
	}
	// Handle non-deleted Operation

	// gets controlled taskGroups items (desired vs actual)
	taskGroups, err := r.reconcileTaskGroups(operation, log)
	if err != nil {
		return err
	}

	err = r.reconcileNormal(operation, taskGroups, log)
	if err != nil {
		return
	}

	// Always reconcile Phase at the end
	r.reconcilePhase(operation)

	return
}

func (r *OperationReconciler) reconcilePause(operation *operatorv1.Operation) {
	// record paused state change, if any
	recordPausedChange(r.recorder, operation, operation.Status.Paused, operation.Spec.Paused)

	// update status with paused setting
	operation.Status.Paused = operation.Spec.Paused
}

func (r *OperationReconciler) reconcileLabels(operation *operatorv1.Operation) {
	if operation.Labels == nil {
		operation.Labels = map[string]string{}
	}
	if _, ok := operation.Labels[operatorv1.OperationNameLabel]; !ok {
		operation.Labels[operatorv1.OperationNameLabel] = operation.Name
	}
	if _, ok := operation.Labels[operatorv1.OperationUIDLabel]; !ok {
		operation.Labels[operatorv1.OperationUIDLabel] = string(uuid.NewUUID())
	}
}

func (r *OperationReconciler) reconcileTaskGroups(operation *operatorv1.Operation, log logr.Logger) (*taskGroupReconcileList, error) {
	// gets all the desired TaskGroup objects for the current operation
	// Nb. this is the domain knowledge encoded into operation implementations
	desired, err := operations.TaskGroupList(operation)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get desired TaskGroup list")
	}

	// gets the current TaskGroup objects related to this Operation
	actual, err := listTaskGroupsByLabels(r.Client, operation.Labels)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list TaskGroup")
	}

	r.Log.Info("reconciling", "desired-TaskGroups", len(desired.Items), "TaskGroups", len(actual.Items))

	// match current and desired TaskGroup, so the controller can determine what is necessary to do next
	taskGroups := reconcileTaskGroups(desired, actual)

	// update replica counters
	operation.Status.Groups = int32(len(taskGroups.all))
	operation.Status.RunningGroups = int32(len(taskGroups.running))
	operation.Status.SucceededGroups = int32(len(taskGroups.completed))
	operation.Status.FailedGroups = int32(len(taskGroups.failed))
	operation.Status.InvalidGroups = int32(len(taskGroups.invalid))

	return taskGroups, nil
}

func (r *OperationReconciler) reconcileNormal(operation *operatorv1.Operation, taskGroups *taskGroupReconcileList, log logr.Logger) error {
	// If the Operation doesn't have finalizer, add it.
	//if !util.Contains(operation.Finalizers, operatorv1.OperationFinalizer) {
	//	operation.Finalizers = append(operation.Finalizers, operatorv1.OperationFinalizer)
	//}

	// if there are TaskGroup not yet completed (pending or running), cleanup error messages (required e.g. after recovery)
	// NB. It is necessary to give priority to running vs errors so the operation controller keeps alive/restarts
	// the DaemonsSet for processing tasks
	if taskGroups.activeTaskGroups() > 0 {
		operation.Status.ResetError()
	} else {
		// if there are invalid combinations (e.g. a TaskGroup without a desired TaskGroup)
		// set the error and stop creating new TaskGroups
		if len(taskGroups.invalid) > 0 {
			// TODO: improve error message
			operation.Status.SetError(
				operatorerrors.NewOperationReconciliationError("something invalid"),
			)
			return nil
		}

		// if there are failed TaskGroup
		// set the error and stop creating new TaskGroups
		if len(taskGroups.failed) > 0 {
			// TODO: improve error message
			operation.Status.SetError(
				operatorerrors.NewOperationReplicaError("something failed"),
			)
			return nil
		}
	}

	// TODO: manage adopt tasks/tasks to be orphaned

	// if nil, set the Operation start time
	if operation.Status.StartTime == nil {
		operation.Status.SetStartTime()

		//TODO: add a signature so we can detect if someone/something changes the operations while it is processed
		return nil
	}

	// if the completed TaskGroup have reached the number of expected TaskGroup, the Operation is completed
	// NB. we are doing this before checking pause because if everything is completed, does not make sense to pause
	if len(taskGroups.completed) == len(taskGroups.all) {
		// NB. we are setting this condition explicitly in order to avoid that the Operation accidentally
		// restarts to create TaskGroup
		operation.Status.SetCompletionTime()
	}

	// if the TaskGroup is paused, return
	if operation.Status.Paused {
		return nil
	}

	// otherwise, proceed creating TaskGroup

	// if there are still TaskGroup to be created
	if len(taskGroups.tobeCreated) > 0 {
		// if there no TaskGroup not yet completed (pending or running)
		if taskGroups.activeTaskGroups() == 0 {
			// create the next TaskGroup in the ordered sequence
			nextTaskGroup := taskGroups.tobeCreated[0].planned
			log.WithValues("task-group", nextTaskGroup.Name).Info("creating task")

			err := r.Client.Create(context.Background(), nextTaskGroup)
			if err != nil {
				return errors.Wrap(err, "Failed to create TaskGroup")
			}
		}
	}

	return nil
}

func (r *OperationReconciler) reconcileDelete(operation *operatorv1.Operation) error {

	// Operation is deleted so remove the finalizer.
	//operation.Finalizers = util.Filter(operation.Finalizers, operatorv1.OperationFinalizer)

	return nil
}

func (r *OperationReconciler) reconcilePhase(operation *operatorv1.Operation) {
	// Set the phase to "deleting" if the deletion timestamp is set.
	if !operation.DeletionTimestamp.IsZero() {
		operation.Status.SetTypedPhase(operatorv1.OperationPhaseDeleted)
		return
	}

	// Set the phase to "failed" if any of Status.ErrorReason or Status.ErrorMessage is not-nil.
	if operation.Status.ErrorReason != nil || operation.Status.ErrorMessage != nil {
		operation.Status.SetTypedPhase(operatorv1.OperationPhaseFailed)
		return
	}

	// Set the phase to "succeeded" if completion date is set.
	if operation.Status.CompletionTime != nil {
		operation.Status.SetTypedPhase(operatorv1.OperationPhaseSucceeded)
		return
	}

	// Set the phase to "paused" if paused set.
	if operation.Status.Paused {
		operation.Status.SetTypedPhase(operatorv1.OperationPhasePaused)
		return
	}

	// Set the phase to "running" if start date is set.
	if operation.Status.StartTime != nil {
		operation.Status.SetTypedPhase(operatorv1.OperationPhaseRunning)
		return
	}

	// Set the phase to "pending" if nil.
	operation.Status.SetTypedPhase(operatorv1.OperationPhasePending)
}
