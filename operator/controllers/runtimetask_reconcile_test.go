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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
)

func Test_newTaskGroupChildProxy(t *testing.T) {
	n1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}

	t1 := operatorv1.RuntimeTask{
		ObjectMeta: metav1.ObjectMeta{
			Name: "task1",
		},
		Spec: operatorv1.RuntimeTaskSpec{
			NodeName: "test",
		},
	}

	type args struct {
		node  *corev1.Node
		tasks []operatorv1.RuntimeTask
	}
	tests := []struct {
		name string
		args args
		want *taskReconcileItem
	}{
		{
			name: "Node and task exist",
			args: args{
				node:  n1,
				tasks: []operatorv1.RuntimeTask{t1},
			},
			want: &taskReconcileItem{
				name:  "test",
				node:  n1,
				tasks: []operatorv1.RuntimeTask{t1},
			},
		},
		{
			name: "Node exists, task does not exist",
			args: args{
				node:  n1,
				tasks: nil,
			},
			want: &taskReconcileItem{
				name:  "test",
				node:  n1,
				tasks: nil,
			},
		},
		{
			name: "Node doest not exist, task exists",
			args: args{
				node:  nil,
				tasks: []operatorv1.RuntimeTask{t1},
			},
			want: &taskReconcileItem{
				name:  "test",
				node:  nil,
				tasks: []operatorv1.RuntimeTask{t1},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newTaskGroupChildProxy(tt.args.node, tt.args.tasks...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newTaskGroupChildProxy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_reconcileTasks(t *testing.T) {
	nodes := []corev1.Node{
		{ // a node with a matching task/pending
			ObjectMeta: metav1.ObjectMeta{
				Name: "node1-a",
			},
		},
		{ // a node with a matching task/running
			ObjectMeta: metav1.ObjectMeta{
				Name: "node1-b",
			},
		},
		{ // a node with a matching task/completed
			ObjectMeta: metav1.ObjectMeta{
				Name: "node1-c",
			},
		},
		{ // a node with a matching task/failed
			ObjectMeta: metav1.ObjectMeta{
				Name: "node1-d",
			},
		},
		{ // a node without a matching task (task to be created)
			ObjectMeta: metav1.ObjectMeta{
				Name: "node2",
			},
		},
		{ // a node with two matching tasks (invalid)
			ObjectMeta: metav1.ObjectMeta{
				Name: "node3",
			},
		},
	}
	tasks := &operatorv1.RuntimeTaskList{
		Items: []operatorv1.RuntimeTask{
			{ // a pending task matching node 1
				ObjectMeta: metav1.ObjectMeta{
					Name: "task1-a",
				},
				Spec: operatorv1.RuntimeTaskSpec{
					NodeName: "node1-a",
				},
			},
			{ // a running task matching node 1
				ObjectMeta: metav1.ObjectMeta{
					Name: "task1-b",
				},
				Spec: operatorv1.RuntimeTaskSpec{
					NodeName: "node1-b",
				},
				Status: operatorv1.RuntimeTaskStatus{
					StartTime: timePtr(metav1.Now()),
				},
			},
			{ // a completed task matching node 1
				ObjectMeta: metav1.ObjectMeta{
					Name: "task1-c",
				},
				Spec: operatorv1.RuntimeTaskSpec{
					NodeName: "node1-c",
				},
				Status: operatorv1.RuntimeTaskStatus{
					StartTime:      timePtr(metav1.Now()),
					CompletionTime: timePtr(metav1.Now()),
				},
			},
			{ // a failed task matching node 1
				ObjectMeta: metav1.ObjectMeta{
					Name: "task1-d",
				},
				Spec: operatorv1.RuntimeTaskSpec{
					NodeName: "node1-d",
				},
				Status: operatorv1.RuntimeTaskStatus{
					StartTime:    timePtr(metav1.Now()),
					ErrorMessage: stringPtr("error"),
				},
			},
			{ // a task matching node 3
				ObjectMeta: metav1.ObjectMeta{
					Name: "task2",
				},
				Spec: operatorv1.RuntimeTaskSpec{
					NodeName: "node3",
				},
			},
			{ // another task matching node 3
				ObjectMeta: metav1.ObjectMeta{
					Name: "task3",
				},
				Spec: operatorv1.RuntimeTaskSpec{
					NodeName: "node3",
				},
			},
			{ // a task not matching any node (invalid)
				ObjectMeta: metav1.ObjectMeta{
					Name: "task4",
				},
				Spec: operatorv1.RuntimeTaskSpec{
					NodeName: "node4",
				},
			},
		},
	}

	got := reconcileTasks(nodes, tasks)

	// All
	want := []string{"node1-a", "node1-b", "node1-c", "node1-d", "node2", "node3", "node4"}
	if got := taskNames(got.all); !reflect.DeepEqual(got, want) {
		t.Errorf("newTaskGroupChildProxy().all = %v, want %v", got, want)
	}

	// pending
	want = []string{"node1-a"}
	if got := taskNames(got.pending); !reflect.DeepEqual(got, want) {
		t.Errorf("newTaskGroupChildProxy().pending = %v, want %v", got, want)
	}

	// running
	want = []string{"node1-b"}
	if got := taskNames(got.running); !reflect.DeepEqual(got, want) {
		t.Errorf("newTaskGroupChildProxy().running = %v, want %v", got, want)
	}

	// completed
	want = []string{"node1-c"}
	if got := taskNames(got.completed); !reflect.DeepEqual(got, want) {
		t.Errorf("newTaskGroupChildProxy().completed = %v, want %v", got, want)
	}

	// failed
	want = []string{"node1-d"}
	if got := taskNames(got.failed); !reflect.DeepEqual(got, want) {
		t.Errorf("newTaskGroupChildProxy().failed = %v, want %v", got, want)
	}

	// tobeCreated
	want = []string{"node2"}
	if got := taskNames(got.tobeCreated); !reflect.DeepEqual(got, want) {
		t.Errorf("newTaskGroupChildProxy().tobeCreated = %v, want %v", got, want)
	}

	// invalid
	want = []string{"node3", "node4"}
	if got := taskNames(got.invalid); !reflect.DeepEqual(got, want) {
		t.Errorf("newTaskGroupChildProxy().invalid = %v, want %v", got, want)
	}
}

func taskNames(got []*taskReconcileItem) []string {
	var actual []string
	for _, a := range got {
		actual = append(actual, a.name)
	}
	return actual
}

func timePtr(t metav1.Time) *metav1.Time {
	return &t
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(s bool) *bool {
	return &s
}
