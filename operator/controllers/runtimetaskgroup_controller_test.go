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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

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
