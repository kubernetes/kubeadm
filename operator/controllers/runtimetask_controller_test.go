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
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"k8s.io/kubeadm/operator/errors"

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
)

func TestRuntimeTaskReconciler_reconcilePhase(t *testing.T) {
	tx := timePtr(metav1.Now())

	type args struct {
		task *operatorv1.RuntimeTask
	}
	type want struct {
		task *operatorv1.RuntimeTask
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Reconcile pending state",
			args: args{
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						Phase: string(operatorv1.RuntimeTaskPhasePending),
					},
				},
			},
		},
		{
			name: "Reconcile running state",
			args: args{
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						StartTime: tx,
					},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						StartTime: tx,
						Phase:     string(operatorv1.RuntimeTaskPhaseRunning),
					},
				},
			},
		},
		{
			name: "Reconcile paused state",
			args: args{
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						StartTime: tx,
						Paused:    true,
					},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						StartTime: tx,
						Paused:    true,
						Phase:     string(operatorv1.RuntimeTaskPhasePaused),
					},
				},
			},
		},
		{
			name: "Reconcile succeeded state",
			args: args{
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						StartTime:      tx,
						CompletionTime: tx,
					},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						StartTime:      tx,
						CompletionTime: tx,
						Phase:          string(operatorv1.RuntimeTaskPhaseSucceeded),
					},
				},
			},
		},
		{
			name: "Reconcile failed state",
			args: args{
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						StartTime:    tx,
						ErrorMessage: stringPtr("error"),
					},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						StartTime:    tx,
						ErrorMessage: stringPtr("error"),
						Phase:        string(operatorv1.RuntimeTaskPhaseFailed),
					},
				},
			},
		},
		{
			name: "Reconcile deleted state",
			args: args{
				task: &operatorv1.RuntimeTask{
					ObjectMeta: metav1.ObjectMeta{
						DeletionTimestamp: tx,
					},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					ObjectMeta: metav1.ObjectMeta{
						DeletionTimestamp: tx,
					},
					Status: operatorv1.RuntimeTaskStatus{
						Phase: string(operatorv1.RuntimeTaskPhaseDeleted),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RuntimeTaskReconciler{}
			r.reconcilePhase(tt.args.task)

			if !reflect.DeepEqual(tt.args.task, tt.want.task) {
				t.Errorf("reconcilePhase() = %v, want %v", tt.args.task, tt.want.task)
			}
		})
	}
}

func TestRuntimeTaskReconciler_reconcilePauseOverride(t *testing.T) {
	type args struct {
		operationPaused bool
		task            *operatorv1.RuntimeTask
	}
	type want struct {
		task   *operatorv1.RuntimeTask
		events int
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Reconcile a Task not paused with an Operation not paused is NOP",
			args: args{
				operationPaused: false,
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						Paused: false,
					},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						Paused: false,
					},
				},
				events: 0,
			},
		},
		{
			name: "Reconcile a Task not paused with an Operation currently paused set pause",
			args: args{
				operationPaused: true,
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						Paused: false,
					},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						Paused: true,
					},
				},
				events: 1,
			},
		},
		{
			name: "Reconcile a Task paused with an Operation currently paused is NOP",
			args: args{
				operationPaused: true,
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						Paused: true,
					},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						Paused: true,
					},
				},
				events: 0,
			},
		},
		{
			name: "Reconcile a Task paused with an Operation currently not paused unset pause",
			args: args{
				operationPaused: false,
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						Paused: true,
					},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						Paused: false,
					},
				},
				events: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := record.NewFakeRecorder(1)

			r := &RuntimeTaskReconciler{
				Client:    nil,
				NodeName:  "",
				Operation: "",
				recorder:  rec,
				Log:       nil,
			}

			r.reconcilePauseOverride(tt.args.operationPaused, tt.args.task)

			if !reflect.DeepEqual(tt.args.task, tt.want.task) {
				t.Errorf("reconcilePauseOverride() = %v, want %v", tt.args.task, tt.want.task)
			}

			if tt.want.events != len(rec.Events) {
				t.Errorf("reconcilePauseOverride() = %v events recorded, want %v events", tt.want.events, len(rec.Events))
			}
		})
	}
}

func TestRuntimeTaskReconciler_reconcileRecovery(t *testing.T) {
	type args struct {
		executionMode operatorv1.OperationExecutionMode
		task          *operatorv1.RuntimeTask
	}
	type want struct {
		ret    bool
		task   *operatorv1.RuntimeTask
		events int
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Reconcile a Task without errors is NOP",
			args: args{
				executionMode: "",
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						Commands: []operatorv1.CommandDescriptor{
							{},
						},
					},
				},
			},
			want: want{
				ret: false,
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						Commands: []operatorv1.CommandDescriptor{
							{},
						},
					},
				},
				events: 0,
			},
		},
		{
			name: "Reconcile a Task without recovery strategy is NOP",
			args: args{
				executionMode: "",
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						Commands: []operatorv1.CommandDescriptor{
							{},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						ErrorReason:  runtimeTaskStatusErrorPtr(errors.RuntimeTaskExecutionError),
						ErrorMessage: stringPtr("error"),
					},
				},
			},
			want: want{
				ret: false,
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						Commands: []operatorv1.CommandDescriptor{
							{},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						ErrorReason:  runtimeTaskStatusErrorPtr(errors.RuntimeTaskExecutionError),
						ErrorMessage: stringPtr("error"),
					},
				},
				events: 0,
			},
		},
		{
			name: "Reconcile a Task using RetryError strategy removes the error (and keep CurrentCommand)",
			args: args{
				executionMode: "",
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						RecoveryMode: string(operatorv1.RuntimeTaskRecoveryRetryingFailedCommandStrategy),
						Commands: []operatorv1.CommandDescriptor{
							{},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						ErrorReason:    runtimeTaskStatusErrorPtr(errors.RuntimeTaskExecutionError),
						ErrorMessage:   stringPtr("error"),
						CurrentCommand: 1,
					},
				},
			},
			want: want{
				ret: true,
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						RecoveryMode: "", // recovery strategy back to empty
						Commands: []operatorv1.CommandDescriptor{
							{},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						ErrorReason:    nil, // error removed
						ErrorMessage:   nil,
						CurrentCommand: 1,
					},
				},
				events: 1,
			},
		},
		{
			name: "Reconcile a Task using SkipError strategy removes the error and moves to the next command",
			args: args{
				executionMode: "",
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						RecoveryMode: string(operatorv1.RuntimeTaskRecoverySkippingFailedCommandStrategy),
						Commands: []operatorv1.CommandDescriptor{
							{},
							{},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						ErrorReason:    runtimeTaskStatusErrorPtr(errors.RuntimeTaskExecutionError),
						ErrorMessage:   stringPtr("error"),
						CurrentCommand: 1, // same command
					},
				},
			},
			want: want{
				ret: true,
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						RecoveryMode: "", // recovery strategy back to empty
						Commands: []operatorv1.CommandDescriptor{
							{},
							{},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						ErrorReason:     nil, // error removed
						ErrorMessage:    nil,
						CurrentCommand:  2, // next command
						CommandProgress: "2/2",
					},
				},
				events: 1,
			},
		},
		{
			name: "Reconcile a Task using SkipError strategy removes the error and moves to the next command + set pause if Mode=Controlled",
			args: args{
				executionMode: operatorv1.OperationExecutionModeControlled,
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						RecoveryMode: string(operatorv1.RuntimeTaskRecoverySkippingFailedCommandStrategy),
						Commands: []operatorv1.CommandDescriptor{
							{},
							{},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						ErrorReason:    runtimeTaskStatusErrorPtr(errors.RuntimeTaskExecutionError),
						ErrorMessage:   stringPtr("error"),
						CurrentCommand: 1,
					},
				},
			},
			want: want{
				ret: true,
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						RecoveryMode: "", // recovery strategy back to empty
						Commands: []operatorv1.CommandDescriptor{
							{},
							{},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						ErrorReason:     nil, // error removed
						ErrorMessage:    nil,
						CurrentCommand:  2, // next command
						CommandProgress: "2/2",
						Paused:          true, // paused
					},
				},
				events: 1,
			},
		},
		{
			name: "Reconcile a Task using SkipError strategy removes the error and set completed if there are no more commands",
			args: args{
				executionMode: operatorv1.OperationExecutionModeControlled,
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						RecoveryMode: string(operatorv1.RuntimeTaskRecoverySkippingFailedCommandStrategy),
						Commands: []operatorv1.CommandDescriptor{
							{},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						ErrorReason:    runtimeTaskStatusErrorPtr(errors.RuntimeTaskExecutionError),
						ErrorMessage:   stringPtr("error"),
						CurrentCommand: 1,
					},
				},
			},
			want: want{
				ret: true,
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						RecoveryMode: "", // recovery strategy back to empty
						Commands: []operatorv1.CommandDescriptor{
							{},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						CompletionTime: timePtr(metav1.Time{}), //using zero as a marker for "whatever time it completes"
						ErrorReason:    nil,                    // error removed
						ErrorMessage:   nil,
						CurrentCommand: 1, // next command
					},
				},
				events: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := record.NewFakeRecorder(1)

			r := &RuntimeTaskReconciler{
				Client:    nil,
				NodeName:  "",
				Operation: "",
				recorder:  rec,
				Log:       log.Log,
			}

			if got := r.reconcileRecovery(tt.args.executionMode, tt.args.task, log.Log); got != tt.want.ret {
				t.Errorf("reconcileRecovery() = %v, want %v", got, tt.want.ret)
			}

			fixupWantTask(tt.want.task, tt.args.task)

			if !reflect.DeepEqual(tt.args.task, tt.want.task) {
				t.Errorf("reconcileRecovery() = %v, want %v", tt.args.task, tt.want.task)
			}

			if tt.want.events != len(rec.Events) {
				t.Errorf("reconcileRecovery() = %v events recorded, want %v events", tt.want.events, len(rec.Events))
			}
		})
	}
}

func TestRuntimeTaskReconciler_reconcileNormal(t *testing.T) {
	type args struct {
		executionMode operatorv1.OperationExecutionMode
		task          *operatorv1.RuntimeTask
	}
	type want struct {
		task   *operatorv1.RuntimeTask
		events int
	}
	tests := []struct {
		name    string
		args    args
		want    want
		wantErr bool
	}{
		{
			name: "Reconcile paused task is a NOP",
			args: args{
				executionMode: "",
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						Paused: true,
					},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					Status: operatorv1.RuntimeTaskStatus{
						Paused: true,
					},
				},
				events: 0,
			},
			wantErr: false,
		},
		{
			name: "Reconcile new task sets start time",
			args: args{
				executionMode: "",
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						Commands: []operatorv1.CommandDescriptor{
							{},
						},
					},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						Commands: []operatorv1.CommandDescriptor{
							{},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						StartTime:       timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
						CurrentCommand:  1,
						CommandProgress: "1/1",
					},
				},
				events: 0,
			},
			wantErr: false,
		},
		{
			name: "Reconcile task already started run commands and move to the next command",
			args: args{
				executionMode: "",
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						Commands: []operatorv1.CommandDescriptor{
							{
								Pass: &operatorv1.PassCommandSpec{},
							},
							{
								Pass: &operatorv1.PassCommandSpec{},
							},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						StartTime:       timePtr(metav1.Now()),
						CurrentCommand:  1,
						CommandProgress: "1/2",
					},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						Commands: []operatorv1.CommandDescriptor{
							{
								Pass: &operatorv1.PassCommandSpec{},
							},
							{
								Pass: &operatorv1.PassCommandSpec{},
							},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						StartTime:       timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
						CurrentCommand:  2,
						CommandProgress: "2/2",
					},
				},
				events: 1,
			},
			wantErr: false,
		},
		{
			name: "Reconcile task already started run commands and move to the next command + pause if operation mode=controlled",
			args: args{
				executionMode: operatorv1.OperationExecutionModeControlled,
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						Commands: []operatorv1.CommandDescriptor{
							{
								Pass: &operatorv1.PassCommandSpec{},
							},
							{
								Pass: &operatorv1.PassCommandSpec{},
							},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						StartTime:       timePtr(metav1.Now()),
						CurrentCommand:  1,
						CommandProgress: "1/2",
					},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						Commands: []operatorv1.CommandDescriptor{
							{
								Pass: &operatorv1.PassCommandSpec{},
							},
							{
								Pass: &operatorv1.PassCommandSpec{},
							},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						StartTime:       timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
						CurrentCommand:  2,
						CommandProgress: "2/2",
						Paused:          true,
					},
				},
				events: 1,
			},
			wantErr: false,
		},
		{
			name: "Reconcile task already started run commands and completes if no more commands",
			args: args{
				executionMode: "",
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						Commands: []operatorv1.CommandDescriptor{
							{
								Pass: &operatorv1.PassCommandSpec{},
							},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						StartTime:       timePtr(metav1.Now()),
						CurrentCommand:  1,
						CommandProgress: "1/1",
					},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						Commands: []operatorv1.CommandDescriptor{
							{
								Pass: &operatorv1.PassCommandSpec{},
							},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						StartTime:       timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
						CurrentCommand:  1,
						CommandProgress: "1/1",
						CompletionTime:  timePtr(metav1.Time{}), //using zero as a marker for "whatever time it completed"
					},
				},
				events: 1,
			},
			wantErr: false,
		},
		{
			name: "Reconcile task handle command failures",
			args: args{
				executionMode: "",
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						Commands: []operatorv1.CommandDescriptor{
							{
								Fail: &operatorv1.FailCommandSpec{},
							},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						StartTime:       timePtr(metav1.Now()),
						CurrentCommand:  1,
						CommandProgress: "1/1",
					},
				},
			},
			want: want{
				task: &operatorv1.RuntimeTask{
					Spec: operatorv1.RuntimeTaskSpec{
						Commands: []operatorv1.CommandDescriptor{
							{
								Fail: &operatorv1.FailCommandSpec{},
							},
						},
					},
					Status: operatorv1.RuntimeTaskStatus{
						StartTime:       timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
						CurrentCommand:  1,
						CommandProgress: "1/1",
						ErrorMessage:    stringPtr("error"), //using error as a marker for "whatever error there is"
					},
				},
				events: 1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := record.NewFakeRecorder(1)

			r := &RuntimeTaskReconciler{
				Client:    nil,
				NodeName:  "",
				Operation: "",
				recorder:  rec,
				Log:       log.Log,
			}
			if err := r.reconcileNormal(tt.args.executionMode, tt.args.task, log.Log); (err != nil) != tt.wantErr {
				t.Errorf("reconcileNormal() error = %v, wantErr %v", err, tt.wantErr)
			}

			fixupWantTask(tt.want.task, tt.args.task)

			if !reflect.DeepEqual(tt.args.task, tt.want.task) {
				t.Errorf("reconcileRecovery() = %v, want %v", tt.args.task, tt.want.task)
			}

			if tt.want.events != len(rec.Events) {
				t.Errorf("reconcileRecovery() = %v events recorded, want %v events", tt.want.events, len(rec.Events))
			}
		})
	}
}

func TestRuntimeTaskReconciler_Reconcile(t *testing.T) {
	type fields struct {
		NodeName  string
		Operation string
		Objs      []runtime.Object
	}
	type args struct {
		req ctrl.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    ctrl.Result
		wantErr bool
	}{
		{
			name:   "Reconcile does nothing if task does not exist",
			fields: fields{},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "foo"}},
			},
			want:    ctrl.Result{},
			wantErr: false,
		},
		{
			name: "Reconcile does nothing if task doesn't target the node the controller is supervising",
			fields: fields{
				NodeName: "foo-node",
				Objs: []runtime.Object{
					&operatorv1.RuntimeTask{
						TypeMeta: metav1.TypeMeta{
							Kind:       "RuntimeTask",
							APIVersion: operatorv1.GroupVersion.String(),
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      "bar-task",
						},
						Spec: operatorv1.RuntimeTaskSpec{
							NodeName: "bar-node",
						},
					},
				},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "bar-task"}},
			},
			want:    ctrl.Result{},
			wantErr: false,
		},
		{
			name: "Reconcile does nothing if the task is already completed",
			fields: fields{
				NodeName: "foo-node",
				Objs: []runtime.Object{
					&operatorv1.RuntimeTask{
						TypeMeta: metav1.TypeMeta{
							Kind:       "RuntimeTask",
							APIVersion: operatorv1.GroupVersion.String(),
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      "foo-task",
						},
						Spec: operatorv1.RuntimeTaskSpec{
							NodeName: "foo-node",
						},
						Status: operatorv1.RuntimeTaskStatus{
							CompletionTime: timePtr(metav1.Now()),
						},
					},
				},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "foo-task"}},
			},
			want:    ctrl.Result{},
			wantErr: false,
		},
		{
			name: "Reconcile fails if failing to retrieve parent taskgroup",
			fields: fields{
				NodeName: "foo-node",
				Objs: []runtime.Object{
					&operatorv1.RuntimeTask{
						TypeMeta: metav1.TypeMeta{
							Kind:       "RuntimeTask",
							APIVersion: operatorv1.GroupVersion.String(),
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      "foo-task",
						},
						Spec: operatorv1.RuntimeTaskSpec{
							NodeName: "foo-node",
						},
					},
				},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "foo-task"}},
			},
			want:    ctrl.Result{},
			wantErr: true,
		},
		{
			name: "Reconcile fails if failing to retrieve parent operation",
			fields: fields{
				NodeName: "foo-node",
				Objs: []runtime.Object{
					&operatorv1.RuntimeTaskGroup{
						TypeMeta: metav1.TypeMeta{
							Kind:       "RuntimeTaskGroup",
							APIVersion: operatorv1.GroupVersion.String(),
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      "foo-taskgroup",
						},
					},
					&operatorv1.RuntimeTask{
						TypeMeta: metav1.TypeMeta{
							Kind:       "RuntimeTask",
							APIVersion: operatorv1.GroupVersion.String(),
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      "foo-task",
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: operatorv1.GroupVersion.String(),
									Kind:       "RuntimeTaskGroup",
									Name:       "foo-taskgroup",
								},
							},
						},
						Spec: operatorv1.RuntimeTaskSpec{
							NodeName: "foo-node",
						},
					},
				},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "foo-task"}},
			},
			want:    ctrl.Result{},
			wantErr: true,
		},
		{
			name: "Reconcile does nothing if task doesn't belong to the operation the controller is supervising",
			fields: fields{
				NodeName:  "foo-node",
				Operation: "foo-operation",
				Objs: []runtime.Object{
					&operatorv1.Operation{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Operation",
							APIVersion: operatorv1.GroupVersion.String(),
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      "bar-operation",
						},
					},
					&operatorv1.RuntimeTaskGroup{
						TypeMeta: metav1.TypeMeta{
							Kind:       "RuntimeTaskGroup",
							APIVersion: operatorv1.GroupVersion.String(),
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      "foo-taskgroup",
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: operatorv1.GroupVersion.String(),
									Kind:       "Operation",
									Name:       "bar-operation",
								},
							},
						},
					},
					&operatorv1.RuntimeTask{
						TypeMeta: metav1.TypeMeta{
							Kind:       "RuntimeTask",
							APIVersion: operatorv1.GroupVersion.String(),
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      "foo-task",
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: operatorv1.GroupVersion.String(),
									Kind:       "RuntimeTaskGroup",
									Name:       "foo-taskgroup",
								},
							},
						},
						Spec: operatorv1.RuntimeTaskSpec{
							NodeName: "foo-node",
						},
					},
				},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "foo-task"}},
			},
			want:    ctrl.Result{},
			wantErr: false,
		},
		{
			name: "Reconcile pass",
			fields: fields{
				NodeName:  "foo-node",
				Operation: "foo-operation",
				Objs: []runtime.Object{
					&operatorv1.Operation{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Operation",
							APIVersion: operatorv1.GroupVersion.String(),
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      "foo-operation",
						},
					},
					&operatorv1.RuntimeTaskGroup{
						TypeMeta: metav1.TypeMeta{
							Kind:       "RuntimeTaskGroup",
							APIVersion: operatorv1.GroupVersion.String(),
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      "foo-taskgroup",
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: operatorv1.GroupVersion.String(),
									Kind:       "Operation",
									Name:       "foo-operation",
								},
							},
						},
					},
					&operatorv1.RuntimeTask{
						TypeMeta: metav1.TypeMeta{
							Kind:       "RuntimeTask",
							APIVersion: operatorv1.GroupVersion.String(),
						},
						ObjectMeta: metav1.ObjectMeta{
							Namespace: "default",
							Name:      "foo-task",
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: operatorv1.GroupVersion.String(),
									Kind:       "RuntimeTaskGroup",
									Name:       "foo-taskgroup",
								},
							},
						},
						Spec: operatorv1.RuntimeTaskSpec{
							NodeName: "foo-node",
						},
					},
				},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "foo-task"}},
			},
			want:    ctrl.Result{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RuntimeTaskReconciler{
				Client:    fake.NewFakeClientWithScheme(setupScheme(), tt.fields.Objs...),
				NodeName:  tt.fields.NodeName,
				Operation: tt.fields.Operation,
				recorder:  record.NewFakeRecorder(1),
				Log:       log.Log,
			}
			got, err := r.Reconcile(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Reconcile() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func fixupWantTask(want *operatorv1.RuntimeTask, got *operatorv1.RuntimeTask) {
	// In case want.StartTime is a marker, replace it with the current CompletionTime
	if want.CreationTimestamp.IsZero() {
		want.CreationTimestamp = got.CreationTimestamp
	}

	// In case want.ErrorMessage is a marker, replace it with the current error
	if want.Status.ErrorMessage != nil && *want.Status.ErrorMessage == "error" && got.Status.ErrorMessage != nil {
		want.Status.ErrorMessage = got.Status.ErrorMessage
		want.Status.ErrorReason = got.Status.ErrorReason
	}

	// In case want.StartTime is a marker, replace it with the current CompletionTime
	if want.Status.StartTime != nil && want.Status.StartTime.IsZero() && got.Status.StartTime != nil {
		want.Status.StartTime = got.Status.StartTime
	}
	// In case want.CompletionTime is a marker, replace it with the current CompletionTime
	if want.Status.CompletionTime != nil && want.Status.CompletionTime.IsZero() && got.Status.CompletionTime != nil {
		want.Status.CompletionTime = got.Status.CompletionTime
	}
}

func runtimeTaskStatusErrorPtr(s errors.RuntimeTaskStatusError) *errors.RuntimeTaskStatusError {
	return &s
}

func setupScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := operatorv1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	return scheme
}
