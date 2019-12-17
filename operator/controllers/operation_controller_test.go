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
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
)

func TestOperatorReconcilePhase(t *testing.T) {
	tx := metav1.Now()
	errMessage := "error"

	tests := []struct {
		name     string
		input    *operatorv1.Operation
		expected *operatorv1.Operation
	}{
		{
			name: "Reconcile pending state",
			input: &operatorv1.Operation{
				Status: operatorv1.OperationStatus{},
			},
			expected: &operatorv1.Operation{
				Status: operatorv1.OperationStatus{
					Phase: string(operatorv1.OperationPhasePending),
				},
			},
		},
		{
			name: "Reconcile running state",
			input: &operatorv1.Operation{
				Status: operatorv1.OperationStatus{
					StartTime: &tx,
				},
			},
			expected: &operatorv1.Operation{
				Status: operatorv1.OperationStatus{
					StartTime: &tx,
					Phase:     string(operatorv1.RuntimeTaskPhaseRunning),
				},
			},
		},
		{
			name: "Reconcile paused state",
			input: &operatorv1.Operation{
				Status: operatorv1.OperationStatus{
					StartTime: &tx,
					Paused:    true,
				},
			},
			expected: &operatorv1.Operation{
				Status: operatorv1.OperationStatus{
					StartTime: &tx,
					Paused:    true,
					Phase:     string(operatorv1.OperationPhasePaused),
				},
			},
		},
		{
			name: "Reconcile succeeded state",
			input: &operatorv1.Operation{
				Status: operatorv1.OperationStatus{
					StartTime:      &tx,
					CompletionTime: &tx,
				},
			},
			expected: &operatorv1.Operation{
				Status: operatorv1.OperationStatus{
					StartTime:      &tx,
					CompletionTime: &tx,
					Phase:          string(operatorv1.OperationPhaseSucceeded),
				},
			},
		},
		{
			name: "Reconcile failed state",
			input: &operatorv1.Operation{
				Status: operatorv1.OperationStatus{
					StartTime:    &tx,
					ErrorMessage: &errMessage,
				},
			},
			expected: &operatorv1.Operation{
				Status: operatorv1.OperationStatus{
					StartTime:    &tx,
					ErrorMessage: &errMessage,
					Phase:        string(operatorv1.OperationPhaseFailed),
				},
			},
		},
		{
			name: "Reconcile deleted state",
			input: &operatorv1.Operation{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &tx,
				},
			},
			expected: &operatorv1.Operation{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &tx,
				},
				Status: operatorv1.OperationStatus{
					Phase: string(operatorv1.OperationPhaseDeleted),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &OperationReconciler{}
			r.reconcilePhase(tt.input)

			if !reflect.DeepEqual(tt.input, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, tt.input)
			}
		})
	}
}

func TestOperatorReconcilePause(t *testing.T) {
	type expected struct {
		operation *operatorv1.Operation
		events    int
	}

	tests := []struct {
		name     string
		input    *operatorv1.Operation
		expected expected
	}{
		{
			name: "Reconcile an Operation not paused with Spec paused",
			input: &operatorv1.Operation{
				Spec: operatorv1.OperationSpec{
					Paused: true,
				},
				Status: operatorv1.OperationStatus{
					Paused: false,
				},
			},
			expected: expected{
				operation: &operatorv1.Operation{
					Spec: operatorv1.OperationSpec{
						Paused: true,
					},
					Status: operatorv1.OperationStatus{
						Paused: true,
					},
				},
				events: 1,
			},
		},
		{
			name: "Reconcile an Operation paused with Spec not paused",
			input: &operatorv1.Operation{
				Spec: operatorv1.OperationSpec{
					Paused: false,
				},
				Status: operatorv1.OperationStatus{
					Paused: true,
				},
			},
			expected: expected{
				operation: &operatorv1.Operation{
					Spec: operatorv1.OperationSpec{
						Paused: false,
					},
					Status: operatorv1.OperationStatus{
						Paused: false,
					},
				},
				events: 1,
			},
		},
		{
			name: "Reconcile an Operation paused with Spec paused",
			input: &operatorv1.Operation{
				Spec: operatorv1.OperationSpec{
					Paused: true,
				},
				Status: operatorv1.OperationStatus{
					Paused: true,
				},
			},
			expected: expected{
				operation: &operatorv1.Operation{
					Spec: operatorv1.OperationSpec{
						Paused: true,
					},
					Status: operatorv1.OperationStatus{
						Paused: true,
					},
				},
				events: 0,
			},
		},
		{
			name: "Reconcile an Operation not paused with Spec not paused",
			input: &operatorv1.Operation{
				Spec: operatorv1.OperationSpec{
					Paused: false,
				},
				Status: operatorv1.OperationStatus{
					Paused: false,
				},
			},
			expected: expected{
				operation: &operatorv1.Operation{
					Spec: operatorv1.OperationSpec{
						Paused: false,
					},
					Status: operatorv1.OperationStatus{
						Paused: false,
					},
				},
				events: 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := record.NewFakeRecorder(1)

			r := &OperationReconciler{
				recorder: rec,
			}

			r.reconcilePause(tt.input)

			if !reflect.DeepEqual(tt.input, tt.expected.operation) {
				t.Errorf("expected %v, got %v", tt.expected.operation, tt.input)
			}

			if tt.expected.events != len(rec.Events) {
				t.Errorf("expected %v, got %v", tt.expected.events, len(rec.Events))
			}
		})
	}
}

func TestOperationReconciler_Reconcile(t *testing.T) {
	type fields struct {
		Objs []runtime.Object
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
			name:   "Reconcile does nothing if operation does not exist",
			fields: fields{},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "foo"}},
			},
			want:    ctrl.Result{},
			wantErr: false,
		},
		{
			name: "Reconcile does nothing if the operation is already completed",
			fields: fields{
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
						Status: operatorv1.OperationStatus{
							CompletionTime: timePtr(metav1.Now()),
						},
					},
				},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "foo-operation"}},
			},
			want:    ctrl.Result{},
			wantErr: false,
		},
		{
			name: "Reconcile pass",
			fields: fields{
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
						Spec: operatorv1.OperationSpec{
							OperatorDescriptor: operatorv1.OperatorDescriptor{
								CustomOperation: &operatorv1.CustomOperationSpec{},
							},
						},
					},
				},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "foo-operation"}},
			},
			want:    ctrl.Result{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &OperationReconciler{
				Client:      fake.NewFakeClientWithScheme(setupScheme(), tt.fields.Objs...),
				AgentImage:  "some-image", //making reconcile operation pass
				MetricsRBAC: false,
				Log:         log.Log,
				recorder:    record.NewFakeRecorder(1),
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

func TestOperationReconciler_reconcileNormal(t *testing.T) {
	type args struct {
		operation  *operatorv1.Operation
		taskGroups *taskGroupReconcileList
	}
	type want struct {
		operation *operatorv1.Operation
	}
	tests := []struct {
		name           string
		args           args
		want           want
		wantTaskGroups int
		wantErr        bool
	}{
		{
			name: "Reconcile sets error if a taskGroup is failed and no taskGroup is active",
			args: args{
				operation: &operatorv1.Operation{},
				taskGroups: &taskGroupReconcileList{
					all: []*taskGroupReconcileItem{
						{},
					},
					failed: []*taskGroupReconcileItem{
						{},
					},
				},
			},
			want: want{
				operation: &operatorv1.Operation{
					Status: operatorv1.OperationStatus{
						ErrorMessage: stringPtr("error"), //using error as a marker for "whatever error was raised"
					},
				},
			},
			wantTaskGroups: 0,
			wantErr:        false,
		},
		{
			name: "Reconcile sets error if a taskGroup is invalid and no taskGroup is active",
			args: args{
				operation: &operatorv1.Operation{},
				taskGroups: &taskGroupReconcileList{
					all: []*taskGroupReconcileItem{
						{},
					},
					invalid: []*taskGroupReconcileItem{
						{},
					},
				},
			},
			want: want{
				operation: &operatorv1.Operation{
					Status: operatorv1.OperationStatus{
						ErrorMessage: stringPtr("error"), //using error as a marker for "whatever error was raised"
					},
				},
			},
			wantTaskGroups: 0,
			wantErr:        false,
		},
		{
			name: "Reconcile set start time",
			args: args{
				operation:  &operatorv1.Operation{},
				taskGroups: &taskGroupReconcileList{},
			},
			want: want{
				operation: &operatorv1.Operation{
					Status: operatorv1.OperationStatus{
						StartTime: timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
					},
				},
			},
			wantTaskGroups: 0,
			wantErr:        false,
		},
		{
			name: "Reconcile reset error if a taskGroup is active",
			args: args{
				operation: &operatorv1.Operation{
					Status: operatorv1.OperationStatus{
						ErrorMessage: stringPtr("error"),
					},
				},
				taskGroups: &taskGroupReconcileList{
					all: []*taskGroupReconcileItem{
						{},
						{},
					},
					running: []*taskGroupReconcileItem{
						{},
					},
					failed: []*taskGroupReconcileItem{ // failed should be ignored if a taskGroup is active
						{},
					},
				},
			},
			want: want{
				operation: &operatorv1.Operation{
					Status: operatorv1.OperationStatus{
						StartTime: timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
					},
				},
			},
			wantTaskGroups: 0,
			wantErr:        false,
		},
		{
			name: "Reconcile set completion time if no more taskGroup to create",
			args: args{
				operation: &operatorv1.Operation{
					Status: operatorv1.OperationStatus{
						StartTime: timePtr(metav1.Now()),
					},
				},
				taskGroups: &taskGroupReconcileList{}, //empty list of nodes -> no more task to create
			},
			want: want{
				operation: &operatorv1.Operation{
					Status: operatorv1.OperationStatus{
						StartTime:      timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
						CompletionTime: timePtr(metav1.Time{}), //using zero as a marker for "whatever time it completed"
					},
				},
			},
			wantTaskGroups: 0,
			wantErr:        false,
		},
		{
			name: "Reconcile do nothing if paused",
			args: args{
				operation: &operatorv1.Operation{
					Status: operatorv1.OperationStatus{
						StartTime: timePtr(metav1.Now()),
						Paused:    true,
					},
				},
				taskGroups: &taskGroupReconcileList{
					all: []*taskGroupReconcileItem{
						{},
					},
					tobeCreated: []*taskGroupReconcileItem{
						{},
					},
				},
			},
			want: want{
				operation: &operatorv1.Operation{
					Status: operatorv1.OperationStatus{
						StartTime: timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
						Paused:    true,
					},
				},
			},
			wantTaskGroups: 0,
			wantErr:        false,
		},
		{
			name: "Reconcile creates a taskGroup if nothing running",
			args: args{
				operation: &operatorv1.Operation{
					Status: operatorv1.OperationStatus{
						StartTime: timePtr(metav1.Now()),
					},
				},
				taskGroups: &taskGroupReconcileList{
					all: []*taskGroupReconcileItem{
						{},
					},
					tobeCreated: []*taskGroupReconcileItem{
						{
							planned: &operatorv1.RuntimeTaskGroup{},
						},
					},
				},
			},
			want: want{
				operation: &operatorv1.Operation{
					Status: operatorv1.OperationStatus{
						StartTime: timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
					},
				},
			},
			wantTaskGroups: 1,
			wantErr:        false,
		},
		{
			name: "Reconcile does not creates a taskGroup if something running",
			args: args{
				operation: &operatorv1.Operation{
					Status: operatorv1.OperationStatus{
						StartTime: timePtr(metav1.Now()),
					},
				},
				taskGroups: &taskGroupReconcileList{
					all: []*taskGroupReconcileItem{
						{},
						{},
					},
					tobeCreated: []*taskGroupReconcileItem{
						{},
					},
					running: []*taskGroupReconcileItem{
						{},
					},
				},
			},
			want: want{
				operation: &operatorv1.Operation{
					Status: operatorv1.OperationStatus{
						StartTime: timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
					},
				},
			},
			wantTaskGroups: 0,
			wantErr:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewFakeClientWithScheme(setupScheme())

			r := &OperationReconciler{
				Client:   c,
				recorder: record.NewFakeRecorder(1),
				Log:      log.Log,
			}
			if err := r.reconcileNormal(tt.args.operation, tt.args.taskGroups, r.Log); (err != nil) != tt.wantErr {
				t.Errorf("reconcileNormal() error = %v, wantErr %v", err, tt.wantErr)
			}

			fixupWantOperation(tt.want.operation, tt.args.operation)

			if !reflect.DeepEqual(tt.args.operation, tt.want.operation) {
				t.Errorf("reconcileNormal() = %v, want %v", tt.args.operation, tt.want.operation)
			}

			taskGroups := &operatorv1.RuntimeTaskGroupList{}
			if err := c.List(context.Background(), taskGroups); err != nil {
				t.Fatalf("List() error = %v", err)
			}

			if len(taskGroups.Items) != tt.wantTaskGroups {
				t.Errorf("reconcileNormal() = %v taskGroups, want %v taskGroups", len(taskGroups.Items), tt.wantTaskGroups)
			}
		})
	}
}

func fixupWantOperation(want *operatorv1.Operation, got *operatorv1.Operation) {
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
