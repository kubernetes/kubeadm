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

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
)

// Reconcile TaskGroups is implemented by matching current and desired TaskGroup, so the controller
// can determine what is necessary to do next

// taskGroupReconcileItem defines match between desired TaskGroup and the corresponding current TaskGroup match.
// supported combinations are:
// - desired existing, current missing (current to be created)
// - desired existing, current existing (current to be operated)
// - desired missing, current existing (invalid)
type taskGroupReconcileItem struct {
	name    string
	planned *operatorv1.RuntimeTaskGroup
	current *operatorv1.RuntimeTaskGroup
}

func newTaskGroupReconcileItem(planned *operatorv1.RuntimeTaskGroup, current *operatorv1.RuntimeTaskGroup) *taskGroupReconcileItem {
	var name string
	if planned != nil {
		name = planned.Name
	} else {
		name = current.Name
	}
	return &taskGroupReconcileItem{
		name:    name,
		planned: planned,
		current: current,
	}
}

type taskGroupReconcileList struct {
	all         []*taskGroupReconcileItem
	invalid     []*taskGroupReconcileItem
	tobeCreated []*taskGroupReconcileItem
	pending     []*taskGroupReconcileItem
	running     []*taskGroupReconcileItem
	completed   []*taskGroupReconcileItem
	failed      []*taskGroupReconcileItem
}

func reconcileTaskGroups(desired *operatorv1.RuntimeTaskGroupList, current *operatorv1.RuntimeTaskGroupList) *taskGroupReconcileList {
	// Build an empty match for each desired TaskGroup
	// N.B. we are storing matches in a Map so we can match desired and current TaskGroup by Name
	matchMap := map[string]*taskGroupReconcileItem{}
	for _, taskGroup := range desired.Items {
		desiredTaskGroup := taskGroup // copies the desired TaskGroup to a local variable in order to avoid it to get overridden at the next iteration
		matchMap[desiredTaskGroup.Name] = newTaskGroupReconcileItem(&desiredTaskGroup, nil)
	}

	// Match the current child objects (TaskGroup) with desired objects (desired TaskGroup).
	for _, taskGroup := range current.Items {
		currentTaskGroup := taskGroup // copies the TaskGroup to a local variable in order to avoid it to get overridden at the next iteration
		// in case a current objects has a corresponding desired object, match them
		// NB. if there are more that one match, we track this, but this is an inconsistency
		// (more that one Task for the same node)
		if v, ok := matchMap[currentTaskGroup.Name]; ok {
			// TODO: might be we want to check if the task was exactly the expected task
			v.current = &currentTaskGroup
			continue
		}

		// in case a current objects does not have desired object, we track this, but this is an inconsistency
		// (a TaskGroup does not matching any desired TaskGroup)
		matchMap[currentTaskGroup.Name] = newTaskGroupReconcileItem(nil, &currentTaskGroup)
	}

	// Transpose the childMap into a list
	matchList := &taskGroupReconcileList{
		all:         []*taskGroupReconcileItem{},
		invalid:     []*taskGroupReconcileItem{},
		tobeCreated: []*taskGroupReconcileItem{},
		pending:     []*taskGroupReconcileItem{},
		running:     []*taskGroupReconcileItem{},
		completed:   []*taskGroupReconcileItem{},
		failed:      []*taskGroupReconcileItem{},
	}

	for _, v := range matchMap {
		matchList.all = append(matchList.all, v)
	}

	// ensure the list is sorted in a predictable way
	sort.Slice(matchList.all, func(i, j int) bool { return matchList.all[i].name < matchList.all[j].name })

	// Build all the derived views, so we can have a quick glance at taskGroups in different states
	matchList.deriveViews()

	return matchList
}

func (a *taskGroupReconcileList) deriveViews() {
	for _, v := range a.all {
		switch {
		case v.planned != nil:
			// If there is not TaskGroup for a desired TaskGroup, the TaskGroup has to be created by this controller
			if v.current == nil {
				a.tobeCreated = append(a.tobeCreated, v)
				continue
			}
			// Failed
			if v.current.Status.ErrorReason != nil || v.current.Status.ErrorMessage != nil {
				a.failed = append(a.failed, v)
				continue
			}
			// Completed
			if v.current.Status.CompletionTime != nil {
				a.completed = append(a.completed, v)
				continue
			}
			// Running (nb. paused TaskGroup fall into this counter)
			if v.current.Status.StartTime != nil {
				a.running = append(a.running, v)
				continue
			}
			// Pending
			a.pending = append(a.pending, v)
		case v.planned == nil:
			a.invalid = append(a.invalid, v)
		}
	}
}

func (a *taskGroupReconcileList) activeTaskGroups() int {
	return len(a.pending) + len(a.running)
}
