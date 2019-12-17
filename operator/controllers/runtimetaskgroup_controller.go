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

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
	operatorerrors "k8s.io/kubeadm/operator/errors"
)

// RuntimeTaskGroupReconciler reconciles a RuntimeTaskGroup object
type RuntimeTaskGroupReconciler struct {
	client.Client
	recorder record.EventRecorder
	Log      logr.Logger
}

// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=operator.kubeadm.x-k8s.io,resources=runtimetaskgroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kubeadm.x-k8s.io,resources=runtimetaskgroups/status,verbs=get;update;patch

// SetupWithManager configures the controller for calling the reconciler
func (r *RuntimeTaskGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var mapFunc handler.ToRequestsFunc = func(o handler.MapObject) []reconcile.Request {
		return operationToTaskGroupRequests(r.Client, o)
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1.RuntimeTaskGroup{}).
		Owns(&operatorv1.RuntimeTask{}). // force reconcile TaskGroup every time one of the owned TaskGroups change
		Watches(                         // force reconcile TaskGroup every time the parent operation changes
			&source.Kind{Type: &operatorv1.Operation{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: mapFunc},
		).
		Complete(r)

	r.recorder = mgr.GetEventRecorderFor("runtime-taskgroup-controller")
	return err
}

// Reconcile a runtimetaskgroup
func (r *RuntimeTaskGroupReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, rerr error) {
	ctx := context.Background()
	log := r.Log.WithValues("task-group", req.NamespacedName)

	// Fetch the TaskGroup instance
	taskgroup := &operatorv1.RuntimeTaskGroup{}
	if err := r.Client.Get(ctx, req.NamespacedName, taskgroup); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Ignore the TaskGroup if it is already completed or failed
	if taskgroup.Status.CompletionTime != nil {
		return ctrl.Result{}, nil
	}

	// Fetch the Operation instance
	operation, err := getOwnerOperation(ctx, r.Client, taskgroup.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(taskgroup, r)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Always attempt to Patch the TaskGroup object and status after each reconciliation.
	defer func() {
		if err := patchHelper.Patch(ctx, taskgroup); err != nil {
			log.Error(err, "failed to patch TaskGroup")
			if rerr == nil {
				rerr = err
			}
		}
	}()

	// Reconcile the TaskGroup
	if err := r.reconcileTaskGroup(operation, taskgroup, log); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *RuntimeTaskGroupReconciler) reconcileTaskGroup(operation *operatorv1.Operation, taskgroup *operatorv1.RuntimeTaskGroup, log logr.Logger) (err error) {
	// gets relevant settings from top level objects
	executionMode := operation.Spec.GetTypedOperationExecutionMode()
	operationPaused := operation.Status.Paused

	// Reconcile paused override from top level objects
	r.reconcilePauseOverride(operationPaused, taskgroup)

	// Handle deleted TaskGroup
	if !taskgroup.DeletionTimestamp.IsZero() {
		err = r.reconcileDelete(taskgroup)
		if err != nil {
			return err
		}
	}
	// Handle non-deleted TaskGroup

	// gets controlled tasks items (desired vs actual)
	tasks, err := r.reconcileTasks(executionMode, taskgroup, log)
	if err != nil {
		return err
	}

	err = r.reconcileNormal(executionMode, taskgroup, tasks, log)
	if err != nil {
		return err
	}

	// Always reconcile Phase at the end
	r.reconcilePhase(taskgroup)

	return nil
}

func (r *RuntimeTaskGroupReconciler) reconcilePauseOverride(operationPaused bool, taskgroup *operatorv1.RuntimeTaskGroup) {
	// record paused override state change, if any
	taskgrouppaused := operationPaused
	recordPausedChange(r.recorder, taskgroup, taskgroup.Status.Paused, taskgrouppaused, "by top level objects")

	// update status with paused override setting from top level objects
	taskgroup.Status.Paused = taskgrouppaused
}

func (r *RuntimeTaskGroupReconciler) reconcileTasks(executionMode operatorv1.OperationExecutionMode, taskgroup *operatorv1.RuntimeTaskGroup, log logr.Logger) (*taskReconcileList, error) {
	// gets all the Node object matching the taskgroup.Spec.NodeSelector
	// those are the Node where the task taskgroup.Spec.Template should be replicated (desired tasks)
	nodes, err := listNodesBySelector(r.Client, &taskgroup.Spec.NodeSelector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list nodes")
	}

	desired := filterNodes(nodes, taskgroup.Spec.GetTypedTaskGroupNodeFilter())

	// gets all the Task objects matching the taskgroup.Spec.Selector.
	// those are the current Task objects controlled by this deployment
	current, err := listTasksBySelector(r.Client, &taskgroup.Spec.Selector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list tasks")
	}

	log.Info("reconciling", "Nodes", len(desired), "Tasks", len(current.Items))

	// match current and desired state, so the controller can determine what is necessary to do next
	tasks := reconcileTasks(desired, current)

	// update replica counters
	taskgroup.Status.Nodes = int32(len(tasks.all))
	taskgroup.Status.RunningNodes = int32(len(tasks.running))
	taskgroup.Status.SucceededNodes = int32(len(tasks.completed))
	taskgroup.Status.FailedNodes = int32(len(tasks.failed))
	taskgroup.Status.InvalidNodes = int32(len(tasks.invalid))

	return tasks, nil
}

func (r *RuntimeTaskGroupReconciler) reconcileNormal(executionMode operatorv1.OperationExecutionMode, taskgroup *operatorv1.RuntimeTaskGroup, tasks *taskReconcileList, log logr.Logger) error {
	// If the TaskGroup doesn't have finalizer, add it.
	//if !util.Contains(taskgroup.Finalizers, operatorv1alpha1.TaskGroupFinalizer) {
	//	taskgroup.Finalizers = append(taskgroup.Finalizers, operatorv1alpha1.TaskGroupFinalizer)
	//}

	// If there are Tasks not yet completed (pending or running), cleanup error messages (required e.g. after recovery)
	// NB. It is necessary to give priority to running vs errors so the operation controller keeps alive/restarts
	// the DaemonsSet for processing tasks
	if tasks.activeTasks() > 0 {
		taskgroup.Status.ResetError()
	} else {
		// if there are invalid combinations (e.g. a Node with more than one Task, or a Task without a Node),
		// set the error and stop creating new Tasks
		if len(tasks.invalid) > 0 {
			taskgroup.Status.SetError(
				operatorerrors.NewRuntimeTaskGroupReconciliationError("something invalid"),
			)
			return nil
		}

		// if there are failed tasks
		// set the error and stop creating new Tasks
		if len(tasks.failed) > 0 {
			taskgroup.Status.SetError(
				operatorerrors.NewRuntimeTaskGroupReplicaError("something failed"),
			)
			return nil
		}
	}

	// TODO: manage adopt tasks/tasks to be orphaned

	// if nil, set the TaskGroup start time
	if taskgroup.Status.StartTime == nil {
		taskgroup.Status.SetStartTime()

		//TODO: add a signature so we can detect if someone/something changes the taskgroup while it is processed

		return nil
	}

	// if the completed Task have reached the number of expected Task, the TaskGroup is completed
	// NB. we are doing this before checking pause because if everything is completed, does not make sense to pause
	if len(tasks.completed) == len(tasks.all) {
		// NB. we are setting this condition explicitly in order to avoid that the taskGroup accidentally
		// restarts to create tasks
		taskgroup.Status.SetCompletionTime()
		return nil
	}

	// if the TaskGroup is paused, return
	if taskgroup.Status.Paused {
		return nil
	}

	// otherwise, proceed creating tasks

	// if there are still Tasks to be created
	if len(tasks.tobeCreated) > 0 {
		//TODO: manage different deployment strategy e.g. parallel

		// if there no existing Tasks not yet completed (pending or running)
		if tasks.activeTasks() == 0 {
			// create a Task for the next node in the ordered sequence
			nextNode := tasks.tobeCreated[0].node.Name
			log.WithValues("node-name", nextNode).Info("creating task")

			err := r.createTasksReplica(executionMode, taskgroup, nextNode)
			if err != nil {
				return errors.Wrap(err, "Failed to create Task replica")
			}
		}
	}

	return nil
}

func (r *RuntimeTaskGroupReconciler) createTasksReplica(executionMode operatorv1.OperationExecutionMode, taskgroup *operatorv1.RuntimeTaskGroup, nodeName string) error {
	r.Log.Info("Creating task replica", "node", nodeName)

	gv := operatorv1.GroupVersion

	paused := false
	if executionMode == operatorv1.OperationExecutionModeControlled {
		paused = true
	}

	task := &operatorv1.RuntimeTask{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RuntimeTask",
			APIVersion: gv.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s-%s", taskgroup.Name, nodeName), //TODO: GeneratedName?
			Namespace:       taskgroup.Namespace,
			Labels:          taskgroup.Spec.Template.Labels,
			Annotations:     taskgroup.Spec.Template.Annotations,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(taskgroup, taskgroup.GroupVersionKind())},
		},
		Spec: operatorv1.RuntimeTaskSpec{
			NodeName: nodeName,
			Commands: taskgroup.Spec.Template.Spec.Commands,
		},
		Status: operatorv1.RuntimeTaskStatus{
			Phase:  string(operatorv1.RuntimeTaskPhasePending),
			Paused: paused,
		},
	}

	return r.Client.Create(context.Background(), task)
}

func (r *RuntimeTaskGroupReconciler) reconcileDelete(taskgroup *operatorv1.RuntimeTaskGroup) error {

	// TaskGroup is deleted so remove the finalizer.
	//taskgroup.Finalizers = util.Filter(taskgroup.Finalizers, operatorv1alpha1.TaskGroupFinalizer)

	return nil
}

func (r *RuntimeTaskGroupReconciler) reconcilePhase(taskgroup *operatorv1.RuntimeTaskGroup) {
	// Set the phase to "deleting" if the deletion timestamp is set.
	if !taskgroup.DeletionTimestamp.IsZero() {
		taskgroup.Status.SetTypedPhase(operatorv1.RuntimeTaskGroupPhaseDeleted)
		return
	}

	// Set the phase to "failed" if any of Status.ErrorReason or Status.ErrorMessage is not nil.
	if taskgroup.Status.ErrorReason != nil || taskgroup.Status.ErrorMessage != nil {
		taskgroup.Status.SetTypedPhase(operatorv1.RuntimeTaskGroupPhaseFailed)
		return
	}

	// Set the phase to "succeeded" if completion date is set.
	if taskgroup.Status.CompletionTime != nil {
		taskgroup.Status.SetTypedPhase(operatorv1.RuntimeTaskGroupPhaseSucceeded)
		return
	}

	// Set the phase to "paused" if paused set.
	if taskgroup.Status.Paused {
		taskgroup.Status.SetTypedPhase(operatorv1.RuntimeTaskGroupPhasePaused)
		return
	}

	// Set the phase to "running" if start date is set.
	if taskgroup.Status.StartTime != nil {
		taskgroup.Status.SetTypedPhase(operatorv1.RuntimeTaskGroupPhaseRunning)
		return
	}

	// Set the phase to "pending".
	taskgroup.Status.SetTypedPhase(operatorv1.RuntimeTaskGroupPhasePending)
}
