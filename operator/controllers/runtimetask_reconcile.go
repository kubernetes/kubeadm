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
	"sort"

	corev1 "k8s.io/api/core/v1"

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
)

// Reconcile Task is implemented by matching current Nodes and desired Task, so the controller
// can determine what is necessary to do next

// taskReconcileItem defines match between desired Task and the corresponding current Node match.
// supported combinations are:
// - desired existing, current missing (current to be created)
// - desired existing, current existing (current to be operated)
// - desired missing, current existing (invalid)

type taskReconcileItem struct {
	name  string
	node  *corev1.Node
	tasks []operatorv1.RuntimeTask
}

func newTaskGroupChildProxy(node *corev1.Node, tasks ...operatorv1.RuntimeTask) *taskReconcileItem {
	var name string
	if node != nil {
		name = node.Name
	} else {
		name = tasks[0].Spec.NodeName
	}
	return &taskReconcileItem{
		name:  name,
		node:  node,
		tasks: tasks,
	}
}

type taskReconcileList struct {
	all         []*taskReconcileItem
	invalid     []*taskReconcileItem
	tobeCreated []*taskReconcileItem
	pending     []*taskReconcileItem
	running     []*taskReconcileItem
	completed   []*taskReconcileItem
	failed      []*taskReconcileItem
}

func reconcileTasks(nodes []corev1.Node, tasks *operatorv1.RuntimeTaskList) *taskReconcileList {
	// Build an empty match for each desired Task (1 for each node)
	// N.B. we are storing matches in a Map so we can match Node and Task by NodeName
	matchMap := map[string]*taskReconcileItem{}
	for _, n := range nodes {
		x := n // copies the node to a local variable in order to avoid it to get overridden at the next iteration
		matchMap[x.Name] = newTaskGroupChildProxy(&x)
	}

	// Match the current Task with desired Task (1 for each node in scope).
	for _, t := range tasks.Items {
		// in case a current task has a corresponding desired task, match them
		// NB. if there are more that one match, we track this, but this is an inconsistency
		// (more that one Task for the same node)
		if v, ok := matchMap[t.Spec.NodeName]; ok {
			// TODO(fabriziopandini): might be we want to check if the task was exactly the expected task
			v.tasks = append(v.tasks, t)
			continue
		}

		// in case a current task does not have desired task, we track this, but this is an inconsistency
		// (a Task does not matching any existing node)
		matchMap[t.Spec.NodeName] = newTaskGroupChildProxy(nil, t)
	}

	// Transpose the matchMap into a list
	matchList := &taskReconcileList{
		all:         []*taskReconcileItem{},
		invalid:     []*taskReconcileItem{},
		tobeCreated: []*taskReconcileItem{},
		pending:     []*taskReconcileItem{},
		running:     []*taskReconcileItem{},
		completed:   []*taskReconcileItem{},
		failed:      []*taskReconcileItem{},
	}

	for _, v := range matchMap {
		matchList.all = append(matchList.all, v)
	}

	// ensure the list is sorted in a predictable way
	sort.Slice(matchList.all, func(i, j int) bool { return matchList.all[i].name < matchList.all[j].name })

	// Build all the derived views, so we can have a quick glance at tasks in different states
	matchList.deriveViews()

	return matchList
}

func (t *taskReconcileList) deriveViews() {
	for _, v := range t.all {
		switch {
		case v.node != nil:
			switch len(v.tasks) {
			case 0:
				// If there is no Task for a Node, the task has to be created by this controller
				t.tobeCreated = append(t.tobeCreated, v)
			case 1:
				// Failed (and not recovering)
				if (v.tasks[0].Status.ErrorReason != nil || v.tasks[0].Status.ErrorMessage != nil) &&
					(v.tasks[0].Spec.GetTypedTaskRecoveryStrategy() == operatorv1.RuntimeTaskRecoveryUnknownStrategy) {
					t.failed = append(t.failed, v)
					continue
				}
				// Completed
				if v.tasks[0].Status.CompletionTime != nil {
					t.completed = append(t.completed, v)
					continue
				}
				// Running (nb. paused Task or recovering Task fall into this counter)
				if v.tasks[0].Status.StartTime != nil {
					t.running = append(t.running, v)
					continue
				}
				// Pending
				t.pending = append(t.pending, v)
			default:
				// if there are more that one Task for the same node, this is an invalid condition
				// NB. in this case it counts as a single replica, even if there are more than one Task
				t.invalid = append(t.invalid, v)
			}
		case v.node == nil:
			// if there is a Task without matching node, this is an invalid condition
			t.invalid = append(t.invalid, v)
		}
	}
}

func (t *taskReconcileList) activeTasks() int {
	return len(t.pending) + len(t.running)
}
