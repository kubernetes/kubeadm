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
