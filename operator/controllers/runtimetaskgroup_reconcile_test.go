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

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
)

func TestNewTaskGroupReconcileItem(t *testing.T) {
	type input struct {
		planned *operatorv1.RuntimeTaskGroup
		current *operatorv1.RuntimeTaskGroup
	}

	planned := &operatorv1.RuntimeTaskGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "planned",
		},
	}
	current := &operatorv1.RuntimeTaskGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "current",
		},
	}

	tests := []struct {
		name     string
		input    input
		expected *taskGroupReconcileItem
	}{
		{
			name: "Planed and current RuntimeTaskGroup exist",
			input: input{
				planned: planned,
				current: current,
			},
			expected: &taskGroupReconcileItem{
				name:    "planned",
				planned: planned,
				current: current,
			},
		},
		{
			name: "Planed RuntimeTaskGroup exists, current RuntimeTaskGroup does not exist",
			input: input{
				planned: planned,
				current: nil,
			},
			expected: &taskGroupReconcileItem{
				name:    "planned",
				planned: planned,
				current: nil,
			},
		},
		{
			name: "Planed RuntimeTaskGroupdoes not exist, current RuntimeTaskGroup exists",
			input: input{
				planned: nil,
				current: current,
			},
			expected: &taskGroupReconcileItem{
				name:    "current",
				planned: nil,
				current: current,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newTaskGroupReconcileItem(tt.input.planned, tt.input.current); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestReconcileTaskGroups(t *testing.T) {
	tx := metav1.Now()
	errMessage := "error"

	desired := &operatorv1.RuntimeTaskGroupList{
		Items: []operatorv1.RuntimeTaskGroup{
			{ // there is not current RuntimeTaskGroup for a desired RuntimeTaskGroup
				ObjectMeta: metav1.ObjectMeta{
					Name: "taskgroup-a",
				},
			},
			{ // a planned RuntimeTaskGroup with a matching current RuntimeTaskGroup/failed
				ObjectMeta: metav1.ObjectMeta{
					Name: "taskgroup-b",
				},
			},
			{ // a planned RuntimeTaskGroup with a matching current RuntimeTaskGroup/completed
				ObjectMeta: metav1.ObjectMeta{
					Name: "taskgroup-c",
				},
			},
			{ // a planned RuntimeTaskGroup with a matching current RuntimeTaskGroup/running
				ObjectMeta: metav1.ObjectMeta{
					Name: "taskgroup-d",
				},
			},
			{ // a planned RuntimeTaskGroup with a matching current RuntimeTaskGroup/pending
				ObjectMeta: metav1.ObjectMeta{
					Name: "taskgroup-e",
				},
			},
		},
	}
	current := &operatorv1.RuntimeTaskGroupList{
		Items: []operatorv1.RuntimeTaskGroup{
			{ // a failed current RuntimeTaskGroup
				ObjectMeta: metav1.ObjectMeta{
					Name: "taskgroup-b",
				},
				Status: operatorv1.RuntimeTaskGroupStatus{
					StartTime:    &tx,
					ErrorMessage: &errMessage,
				},
			},
			{ // a completed current RuntimeTaskGroup
				ObjectMeta: metav1.ObjectMeta{
					Name: "taskgroup-c",
				},
				Status: operatorv1.RuntimeTaskGroupStatus{
					StartTime:      &tx,
					CompletionTime: &tx,
				},
			},
			{ // a running current RuntimeTaskGroup
				ObjectMeta: metav1.ObjectMeta{
					Name: "taskgroup-d",
				},
				Status: operatorv1.RuntimeTaskGroupStatus{
					StartTime: &tx,
				},
			},
			{ // a pending current RuntimeTaskGroup
				ObjectMeta: metav1.ObjectMeta{
					Name: "taskgroup-e",
				},
			},
			{ // a current RuntimeTaskGroup not matching any desired RuntimeTaskGroup
				ObjectMeta: metav1.ObjectMeta{
					Name: "taskgroup-f",
				},
			},
		},
	}

	got := reconcileTaskGroups(desired, current)

	// All
	expected := []string{"taskgroup-a", "taskgroup-b", "taskgroup-c", "taskgroup-d", "taskgroup-e", "taskgroup-f"}
	if got := taskGroupNames(got.all); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}

	// pending
	expected = []string{"taskgroup-e"}
	if got := taskGroupNames(got.pending); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}

	// running
	expected = []string{"taskgroup-d"}
	if got := taskGroupNames(got.running); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}

	// completed
	expected = []string{"taskgroup-c"}
	if got := taskGroupNames(got.completed); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}

	// failed
	expected = []string{"taskgroup-b"}
	if got := taskGroupNames(got.failed); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}

	// tobeCreated
	expected = []string{"taskgroup-a"}
	if got := taskGroupNames(got.tobeCreated); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}

	// invalid
	expected = []string{"taskgroup-f"}
	if got := taskGroupNames(got.invalid); !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func taskGroupNames(got []*taskGroupReconcileItem) []string {
	var actual []string
	for _, a := range got {
		actual = append(actual, a.name)
	}
	return actual
}
