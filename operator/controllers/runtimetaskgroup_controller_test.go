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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
)

func TestRuntimeTaskGorupReconcilePhase(t *testing.T) {
	tx := metav1.Now()
	errMessage := "error"

	tests := []struct {
		name     string
		input    *operatorv1.RuntimeTaskGroup
		expected *operatorv1.RuntimeTaskGroup
	}{
		{
			name: "Reconcile pending state",
			input: &operatorv1.RuntimeTaskGroup{
				Status: operatorv1.RuntimeTaskGroupStatus{},
			},
			expected: &operatorv1.RuntimeTaskGroup{
				Status: operatorv1.RuntimeTaskGroupStatus{
					Phase: string(operatorv1.RuntimeTaskGroupPhasePending),
				},
			},
		},
		{
			name: "Reconcile running state",
			input: &operatorv1.RuntimeTaskGroup{
				Status: operatorv1.RuntimeTaskGroupStatus{
					StartTime: &tx,
				},
			},
			expected: &operatorv1.RuntimeTaskGroup{
				Status: operatorv1.RuntimeTaskGroupStatus{
					StartTime: &tx,
					Phase:     string(operatorv1.RuntimeTaskGroupPhaseRunning),
				},
			},
		},
		{
			name: "Reconcile paused state",
			input: &operatorv1.RuntimeTaskGroup{
				Status: operatorv1.RuntimeTaskGroupStatus{
					StartTime: &tx,
					Paused:    true,
				},
			},
			expected: &operatorv1.RuntimeTaskGroup{
				Status: operatorv1.RuntimeTaskGroupStatus{
					StartTime: &tx,
					Paused:    true,
					Phase:     string(operatorv1.RuntimeTaskGroupPhasePaused),
				},
			},
		},
		{
			name: "Reconcile succeeded state",
			input: &operatorv1.RuntimeTaskGroup{
				Status: operatorv1.RuntimeTaskGroupStatus{
					StartTime:      &tx,
					CompletionTime: &tx,
				},
			},
			expected: &operatorv1.RuntimeTaskGroup{
				Status: operatorv1.RuntimeTaskGroupStatus{
					StartTime:      &tx,
					CompletionTime: &tx,
					Phase:          string(operatorv1.RuntimeTaskGroupPhaseSucceeded),
				},
			},
		},
		{
			name: "Reconcile failed state",
			input: &operatorv1.RuntimeTaskGroup{
				Status: operatorv1.RuntimeTaskGroupStatus{
					StartTime:    &tx,
					ErrorMessage: &errMessage,
				},
			},
			expected: &operatorv1.RuntimeTaskGroup{
				Status: operatorv1.RuntimeTaskGroupStatus{
					StartTime:    &tx,
					ErrorMessage: &errMessage,
					Phase:        string(operatorv1.RuntimeTaskGroupPhaseFailed),
				},
			},
		},
		{
			name: "Reconcile deleted state",
			input: &operatorv1.RuntimeTaskGroup{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &tx,
				},
			},
			expected: &operatorv1.RuntimeTaskGroup{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &tx,
				},
				Status: operatorv1.RuntimeTaskGroupStatus{
					Phase: string(operatorv1.RuntimeTaskGroupPhaseDeleted),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RuntimeTaskGroupReconciler{}
			r.reconcilePhase(tt.input)

			if !reflect.DeepEqual(tt.input, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, tt.input)
			}
		})
	}
}

func TestRuntimeTaskGorupReconcilePauseOverride(t *testing.T) {
	type input struct {
		operationPaused bool
		taskgroup       *operatorv1.RuntimeTaskGroup
	}
	type expected struct {
		taskgroup *operatorv1.RuntimeTaskGroup
		events    int
	}

	tests := []struct {
		name     string
		input    input
		expected expected
	}{
		{
			name: "Reconcile a RuntimeTaskGroup not paused with an Operation not paused",
			input: input{
				operationPaused: false,
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						Paused: false,
					},
				},
			},
			expected: expected{
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						Paused: false,
					},
				},
				events: 0,
			},
		},
		{
			name: "Reconcile a RuntimeTaskGroup not paused with an Operation paused",
			input: input{
				operationPaused: true,
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						Paused: false,
					},
				},
			},
			expected: expected{
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						Paused: true,
					},
				},
				events: 1,
			},
		},
		{
			name: "Reconcile a RuntimeTaskGroup paused with an Operation paused",
			input: input{
				operationPaused: true,
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						Paused: true,
					},
				},
			},
			expected: expected{
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						Paused: true,
					},
				},
				events: 0,
			},
		},
		{
			name: "Reconcile a RuntimeTaskGroup paused with an Operation not paused",
			input: input{
				operationPaused: false,
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						Paused: true,
					},
				},
			},
			expected: expected{
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
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

			r := &RuntimeTaskGroupReconciler{
				recorder: rec,
			}

			r.reconcilePauseOverride(tt.input.operationPaused, tt.input.taskgroup)

			if !reflect.DeepEqual(tt.input.taskgroup, tt.expected.taskgroup) {
				t.Errorf("expected %v, got %v", tt.expected.taskgroup, tt.input.taskgroup)
			}

			if tt.expected.events != len(rec.Events) {
				t.Errorf("expected %v, got %v", tt.expected.events, len(rec.Events))
			}
		})
	}
}

func TestRuntimeTaskGroupReconciler_createTasksReplica(t *testing.T) {
	type args struct {
		executionMode operatorv1.OperationExecutionMode
		taskgroup     *operatorv1.RuntimeTaskGroup
		nodeName      string
	}
	tests := []struct {
		name    string
		args    args
		want    *operatorv1.RuntimeTask
		wantErr bool
	}{
		{
			name: "Create a task",
			args: args{
				executionMode: operatorv1.OperationExecutionModeAuto,
				taskgroup: &operatorv1.RuntimeTaskGroup{
					TypeMeta: metav1.TypeMeta{
						Kind:       "RuntimeTaskGroup",
						APIVersion: operatorv1.GroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "foo-taskgroup",
					},
					Spec: operatorv1.RuntimeTaskGroupSpec{
						Template: operatorv1.RuntimeTaskTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels:      map[string]string{"foo-label": "foo"},
								Annotations: map[string]string{"foo-annotation": "foo"},
							},
							Spec: operatorv1.RuntimeTaskSpec{
								Commands: []operatorv1.CommandDescriptor{
									{},
								},
							},
						},
					},
				},
				nodeName: "foo-node",
			},
			want: &operatorv1.RuntimeTask{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RuntimeTask",
					APIVersion: operatorv1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "foo-taskgroup-foo-node",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         operatorv1.GroupVersion.String(),
							Kind:               "RuntimeTaskGroup",
							Name:               "foo-taskgroup",
							UID:                "",
							Controller:         boolPtr(true),
							BlockOwnerDeletion: boolPtr(true),
						},
					},
					CreationTimestamp: metav1.Time{}, //using zero as a marker for "whatever time it was created"
					Labels:            map[string]string{"foo-label": "foo"},
					Annotations:       map[string]string{"foo-annotation": "foo"},
				},
				Spec: operatorv1.RuntimeTaskSpec{
					NodeName: "foo-node",
					Commands: []operatorv1.CommandDescriptor{
						{},
					},
				},
				Status: operatorv1.RuntimeTaskStatus{
					Phase:  string(operatorv1.RuntimeTaskPhasePending),
					Paused: false,
				},
			},
			wantErr: false,
		},
		{
			name: "Create a paused task if execution mode=Controlled",
			args: args{
				executionMode: operatorv1.OperationExecutionModeControlled,
				taskgroup: &operatorv1.RuntimeTaskGroup{
					TypeMeta: metav1.TypeMeta{
						Kind:       "RuntimeTaskGroup",
						APIVersion: operatorv1.GroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      "foo-taskgroup",
					},
					Spec: operatorv1.RuntimeTaskGroupSpec{
						Template: operatorv1.RuntimeTaskTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels:      map[string]string{"foo-label": "foo"},
								Annotations: map[string]string{"foo-annotation": "foo"},
							},
							Spec: operatorv1.RuntimeTaskSpec{
								Commands: []operatorv1.CommandDescriptor{
									{},
								},
							},
						},
					},
				},
				nodeName: "foo-node",
			},
			want: &operatorv1.RuntimeTask{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RuntimeTask",
					APIVersion: operatorv1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "foo-taskgroup-foo-node",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         operatorv1.GroupVersion.String(),
							Kind:               "RuntimeTaskGroup",
							Name:               "foo-taskgroup",
							UID:                "",
							Controller:         boolPtr(true),
							BlockOwnerDeletion: boolPtr(true),
						},
					},
					CreationTimestamp: metav1.Time{}, //using zero as a marker for "whatever time it was created"
					Labels:            map[string]string{"foo-label": "foo"},
					Annotations:       map[string]string{"foo-annotation": "foo"},
				},
				Spec: operatorv1.RuntimeTaskSpec{
					NodeName: "foo-node",
					Commands: []operatorv1.CommandDescriptor{
						{},
					},
				},
				Status: operatorv1.RuntimeTaskStatus{
					Phase:  string(operatorv1.RuntimeTaskPhasePending),
					Paused: true,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RuntimeTaskGroupReconciler{
				Client: fake.NewFakeClientWithScheme(setupScheme()),
				Log:    log.Log,
			}
			if err := r.createTasksReplica(tt.args.executionMode, tt.args.taskgroup, tt.args.nodeName); (err != nil) != tt.wantErr {
				t.Errorf("createTasksReplica() error = %v, wantErr %v", err, tt.wantErr)
			}

			got := &operatorv1.RuntimeTask{}
			key := client.ObjectKey{
				Namespace: tt.want.Namespace,
				Name:      tt.want.Name,
			}
			if err := r.Client.Get(context.Background(), key, got); err != nil {
				t.Errorf("Get() error = %v", err)
				return
			}

			fixupWantTask(tt.want, got)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createTasksReplica() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRuntimeTaskGroupReconciler_reconcileNormal(t *testing.T) {
	type args struct {
		executionMode operatorv1.OperationExecutionMode
		taskgroup     *operatorv1.RuntimeTaskGroup
		tasks         *taskReconcileList
	}
	type want struct {
		taskgroup *operatorv1.RuntimeTaskGroup
	}
	tests := []struct {
		name      string
		args      args
		want      want
		wantTasks int
		wantErr   bool
	}{
		{
			name: "Reconcile sets error if a task is failed and no task is active",
			args: args{
				executionMode: "",
				taskgroup:     &operatorv1.RuntimeTaskGroup{},
				tasks: &taskReconcileList{
					all: []*taskReconcileItem{
						{},
					},
					failed: []*taskReconcileItem{
						{},
					},
				},
			},
			want: want{
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						ErrorMessage: stringPtr("error"), //using error as a marker for "whatever error was raised"
					},
				},
			},
			wantTasks: 0,
			wantErr:   false,
		},
		{
			name: "Reconcile sets error if a task is invalid and no task is active",
			args: args{
				executionMode: "",
				taskgroup:     &operatorv1.RuntimeTaskGroup{},
				tasks: &taskReconcileList{
					all: []*taskReconcileItem{
						{},
					},
					invalid: []*taskReconcileItem{
						{},
					},
				},
			},
			want: want{
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						ErrorMessage: stringPtr("error"), //using error as a marker for "whatever error was raised"
					},
				},
			},
			wantTasks: 0,
			wantErr:   false,
		},
		{
			name: "Reconcile set start time",
			args: args{
				executionMode: "",
				taskgroup:     &operatorv1.RuntimeTaskGroup{},
				tasks:         &taskReconcileList{},
			},
			want: want{
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						StartTime: timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
					},
				},
			},
			wantTasks: 0,
			wantErr:   false,
		},
		{
			name: "Reconcile reset error if a task is active",
			args: args{
				executionMode: "",
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						ErrorMessage: stringPtr("error"),
					},
				},
				tasks: &taskReconcileList{
					all: []*taskReconcileItem{
						{},
						{},
					},
					running: []*taskReconcileItem{
						{},
					},
					failed: []*taskReconcileItem{ // failed should be ignored if a task is active
						{},
					},
				},
			},
			want: want{
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						StartTime: timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
					},
				},
			},
			wantTasks: 0,
			wantErr:   false,
		},
		{
			name: "Reconcile set completion time if no more task to create",
			args: args{
				executionMode: "",
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						StartTime: timePtr(metav1.Now()),
					},
				},
				tasks: &taskReconcileList{}, //empty list of nodes -> no more task to create
			},
			want: want{
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						StartTime:      timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
						CompletionTime: timePtr(metav1.Time{}), //using zero as a marker for "whatever time it completed"
					},
				},
			},
			wantTasks: 0,
			wantErr:   false,
		},
		{
			name: "Reconcile do nothing if paused",
			args: args{
				executionMode: "",
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						StartTime: timePtr(metav1.Now()),
						Paused:    true,
					},
				},
				tasks: &taskReconcileList{
					all: []*taskReconcileItem{
						{},
					},
					tobeCreated: []*taskReconcileItem{
						{},
					},
				},
			},
			want: want{
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						StartTime: timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
						Paused:    true,
					},
				},
			},
			wantTasks: 0,
			wantErr:   false,
		},
		{
			name: "Reconcile creates a task if nothing running",
			args: args{
				executionMode: "",
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						StartTime: timePtr(metav1.Now()),
					},
				},
				tasks: &taskReconcileList{
					all: []*taskReconcileItem{
						{},
					},
					tobeCreated: []*taskReconcileItem{
						{
							node: &corev1.Node{
								ObjectMeta: metav1.ObjectMeta{
									Name: "foo",
								},
							},
						},
					},
				},
			},
			want: want{
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						StartTime: timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
					},
				},
			},
			wantTasks: 1,
			wantErr:   false,
		},
		{
			name: "Reconcile does not creates a task if something running",
			args: args{
				executionMode: "",
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						StartTime: timePtr(metav1.Now()),
					},
				},
				tasks: &taskReconcileList{
					all: []*taskReconcileItem{
						{},
						{},
					},
					tobeCreated: []*taskReconcileItem{
						{},
					},
					running: []*taskReconcileItem{
						{},
					},
				},
			},
			want: want{
				taskgroup: &operatorv1.RuntimeTaskGroup{
					Status: operatorv1.RuntimeTaskGroupStatus{
						StartTime: timePtr(metav1.Time{}), //using zero as a marker for "whatever time it started"
					},
				},
			},
			wantTasks: 0,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewFakeClientWithScheme(setupScheme())

			r := &RuntimeTaskGroupReconciler{
				Client:   c,
				recorder: record.NewFakeRecorder(1),
				Log:      log.Log,
			}
			if err := r.reconcileNormal(tt.args.executionMode, tt.args.taskgroup, tt.args.tasks, r.Log); (err != nil) != tt.wantErr {
				t.Errorf("reconcileNormal() error = %v, wantErr %v", err, tt.wantErr)
			}

			fixupWantTaskGroup(tt.want.taskgroup, tt.args.taskgroup)

			if !reflect.DeepEqual(tt.args.taskgroup, tt.want.taskgroup) {
				t.Errorf("reconcileNormal() = %v, want %v", tt.args.taskgroup, tt.want.taskgroup)
			}

			tasks := &operatorv1.RuntimeTaskList{}
			if err := c.List(context.Background(), tasks); err != nil {
				t.Fatalf("List() error = %v", err)
			}

			if len(tasks.Items) != tt.wantTasks {
				t.Errorf("reconcileNormal() = %v tasks, want %v tasks", len(tasks.Items), tt.wantTasks)
			}
		})
	}
}

func fixupWantTaskGroup(want *operatorv1.RuntimeTaskGroup, got *operatorv1.RuntimeTaskGroup) {
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
