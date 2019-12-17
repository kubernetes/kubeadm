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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
	commandimpl "k8s.io/kubeadm/operator/commands"
	operatorerrors "k8s.io/kubeadm/operator/errors"
)

// RuntimeTaskReconciler reconciles a RuntimeTask object
type RuntimeTaskReconciler struct {
	client.Client
	NodeName  string
	Operation string
	recorder  record.EventRecorder
	Log       logr.Logger
}

// +kubebuilder:rbac:groups=operator.kubeadm.x-k8s.io,resources=runtimetasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kubeadm.x-k8s.io,resources=runtimetasks/status,verbs=get;update;patch

// SetupWithManager configures the controller for calling the reconciler
func (r *RuntimeTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var mapFunc handler.ToRequestsFunc = func(o handler.MapObject) []reconcile.Request {
		return taskGroupToTaskRequests(r.Client, o)
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1.RuntimeTask{}).
		Watches( // force reconcile Task every time the parent TaskGroup changes
			&source.Kind{Type: &operatorv1.RuntimeTaskGroup{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: mapFunc},
		).
		Complete(r)

	r.recorder = mgr.GetEventRecorderFor("runtime-task-controller")
	return err
}

// Reconcile a runtimetask
func (r *RuntimeTaskReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, rerr error) {
	ctx := context.Background()
	log := r.Log.WithValues("task", req.NamespacedName)

	// Fetch the Task instance
	task := &operatorv1.RuntimeTask{}
	if err := r.Client.Get(ctx, req.NamespacedName, task); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Ignore the Task if it doesn't target the node the controller is supervising
	if task.Spec.NodeName != r.NodeName {
		return ctrl.Result{}, nil
	}

	// Ignore the Task if it is already completed or failed
	if task.Status.CompletionTime != nil {
		return ctrl.Result{}, nil
	}

	// Fetch the parent TaskGroup instance
	taskgroup, err := getOwnerTaskGroup(ctx, r.Client, task.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the parent Operation instance
	operation, err := getOwnerOperation(ctx, r.Client, taskgroup.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}

	// If the controller is set to manage Task for a specific operation, ignore everything else
	if r.Operation != operation.Name {
		return ctrl.Result{}, nil
	}

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(task, r)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Always attempt to Patch the Task object and status after each reconciliation.
	defer func() {
		if err := patchHelper.Patch(ctx, task); err != nil {
			log.Error(err, "failed to patch Task")
			if rerr == nil {
				rerr = err
			}
		}
	}()

	// Reconcile the Task
	if err := r.reconcileTask(operation, taskgroup, task, log); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *RuntimeTaskReconciler) reconcileTask(operation *operatorv1.Operation, taskgroup *operatorv1.RuntimeTaskGroup, task *operatorv1.RuntimeTask, log logr.Logger) (err error) {
	// gets relevant settings from top level objects
	executionMode := operation.Spec.GetTypedOperationExecutionMode()
	operationPaused := operation.Status.Paused

	// Reconcile recovery from errors
	recovered := r.reconcileRecovery(executionMode, task, log)

	// Reconcile paused override from top level objects
	r.reconcilePauseOverride(operationPaused, task)

	// Handle deleted Task
	if !task.DeletionTimestamp.IsZero() {
		err = r.reconcileDelete(task)
		if err != nil {
			return
		}
	}
	// Handle non-deleted/non-recovered Task
	// NB. in case of a task recovered from error, we are forcing another reconcile before actually
	// executing the next command so the user get evidence of what is happening
	if !recovered {
		err = r.reconcileNormal(executionMode, task, log)
		if err != nil {
			return
		}
	}

	// Always reconcile Task Phase at the end
	r.reconcilePhase(task)

	return
}

func (r *RuntimeTaskReconciler) reconcileRecovery(executionMode operatorv1.OperationExecutionMode, task *operatorv1.RuntimeTask, log logr.Logger) bool {
	// if there is no error, return
	if task.Status.ErrorReason == nil && task.Status.ErrorMessage == nil {
		return false
	}

	// if there is no error recovery strategy, return
	if task.Spec.GetTypedTaskRecoveryStrategy() == operatorv1.RuntimeTaskRecoveryUnknownStrategy {
		return false
	}

	switch task.Spec.GetTypedTaskRecoveryStrategy() {
	case operatorv1.RuntimeTaskRecoveryRetryingFailedCommandStrategy:
		log.WithValues("command", task.Status.CurrentCommand).Info("Retrying command after failure")
		r.recorder.Event(task, corev1.EventTypeNormal, "TaskErrorRetry", fmt.Sprintf("Retrying command %d after failure", task.Status.CurrentCommand))
	case operatorv1.RuntimeTaskRecoverySkippingFailedCommandStrategy:
		log.WithValues("command", task.Status.CurrentCommand).Info("Skipping command after failure")
		r.recorder.Event(task, corev1.EventTypeNormal, "TaskErrorSkip", fmt.Sprintf("Skipping command %d after failure", task.Status.CurrentCommand))

		// if all the commands are done, set the Task completion time
		if int(task.Status.CurrentCommand) >= len(task.Spec.Commands) {
			task.Status.SetCompletionTime()
		} else {
			// Move to the next command
			task.Status.NextCurrentCommand(task.Spec.Commands)
			if executionMode == operatorv1.OperationExecutionModeControlled {
				task.Status.Paused = true
			}
		}

	default:
		//TODO: error (if possible do validation before getting here)
	}

	// Reset the error
	task.Status.ResetError()

	// Reset the recovery mode so the user can choose again how to proceed at the next error
	task.Spec.RecoveryMode = ""

	return true
}

func (r *RuntimeTaskReconciler) reconcilePauseOverride(operationPaused bool, task *operatorv1.RuntimeTask) {
	// record paused override state change, if any
	pausedOverride := operationPaused
	recordPausedChange(r.recorder, task, task.Status.Paused, pausedOverride, "by top level objects")

	// update status with paused override setting from top level objects
	task.Status.Paused = pausedOverride
}

func (r *RuntimeTaskReconciler) reconcileNormal(executionMode operatorv1.OperationExecutionMode, task *operatorv1.RuntimeTask, log logr.Logger) error {
	// If the Task doesn't have finalizer, add it.
	//if !util.Contains(task.Finalizers, operatorv1.RuntimeTaskFinalizer) {
	//	task.Finalizers = append(task.Finalizers, operatorv1.RuntimeTaskFinalizer)
	//}

	// if  higher level object are paused, return
	if task.Status.Paused {
		return nil
	}

	// if nil, set the Task start time, initialize CurrentCommand and return
	// NB. we are returning here so the object get updated reporting start condition
	// before actual execution starts
	if task.Status.StartTime == nil {
		task.Status.SetStartTime()
		task.Status.NextCurrentCommand(task.Spec.Commands)
		return nil
	}

	// Proceed with the current command execution

	if executionMode == operatorv1.OperationExecutionModeDryRun {
		// if dry running wait for an arbitrary delay so the user will get a better perception of the Task execution order
		time.Sleep(3 * time.Second)
	} else {
		// else we should execute the CurrentCommand
		log.WithValues("command", task.Status.CurrentCommand).Info("running command")

		// transpose CurrentCommand (1 based) to index (0 based) and check index out of range
		index := int(task.Status.CurrentCommand) - 1
		if index < 0 || index >= len(task.Spec.Commands) {
			task.Status.SetError(
				operatorerrors.NewRuntimeTaskIndexOutOfRangeError("command with index %d does not exists for task %s", index, task.Name),
			)
		}

		// execute the command
		err := commandimpl.RunCommand(&task.Spec.Commands[index], log)

		// if the command returned an error, return
		if err != nil {
			log.WithValues("command", task.Status.CurrentCommand).WithValues("error", fmt.Sprintf("%+v", err)).Info("command failed")
			r.recorder.Event(task, corev1.EventTypeWarning, "CommandError", fmt.Sprintf("Command %d execution failed: %s", task.Status.CurrentCommand, fmt.Sprintf("%+v", err)))
			task.Status.SetError(
				operatorerrors.NewRuntimeTaskExecutionError("error executing command number %d for task %s: %+v", task.Status.CurrentCommand, task.Name, err),
			)
			return nil
		}

		log.WithValues("command", task.Status.CurrentCommand).Info("command completed")
		r.recorder.Event(task, corev1.EventTypeNormal, "CommandCompleted", fmt.Sprintf("Command %d execution completed", task.Status.CurrentCommand))
	}

	// if all the commands are done, set the Task completion time and return
	if int(task.Status.CurrentCommand) >= len(task.Spec.Commands) {
		task.Status.SetCompletionTime()
		return nil
	}

	// Otherwise, move to the next command
	task.Status.NextCurrentCommand(task.Spec.Commands)
	if executionMode == operatorv1.OperationExecutionModeControlled {
		task.Status.Paused = true
	}
	return nil
}

func (r *RuntimeTaskReconciler) reconcileDelete(task *operatorv1.RuntimeTask) error {

	// Task is deleted so remove the finalizer.
	//task.Finalizers = util.Filter(task.Finalizers, operatorv1.RuntimeTaskFinalizer)

	return nil
}

func (r *RuntimeTaskReconciler) reconcilePhase(task *operatorv1.RuntimeTask) {
	// Set the phase to "deleting" if the deletion timestamp is set.
	if !task.DeletionTimestamp.IsZero() {
		task.Status.SetTypedPhase(operatorv1.RuntimeTaskPhaseDeleted)
		return
	}

	// Set the phase to "failed" if any of Status.ErrorReason or Status.ErrorMessage is not-nil.
	if task.Status.ErrorReason != nil || task.Status.ErrorMessage != nil {
		task.Status.SetTypedPhase(operatorv1.RuntimeTaskPhaseFailed)
		return
	}

	// Set the phase to "succeeded" if completion date is set.
	if task.Status.CompletionTime != nil {
		task.Status.SetTypedPhase(operatorv1.RuntimeTaskPhaseSucceeded)
		return
	}

	// Set the phase to "paused" if paused is set.
	if task.Status.Paused {
		task.Status.SetTypedPhase(operatorv1.RuntimeTaskPhasePaused)
		return
	}

	// Set the phase to "running" if start date is set.
	if task.Status.StartTime != nil {
		task.Status.SetTypedPhase(operatorv1.RuntimeTaskPhaseRunning)
		return
	}

	// Set the phase to "pending" if nil.
	task.Status.SetTypedPhase(operatorv1.RuntimeTaskPhasePending)
}
