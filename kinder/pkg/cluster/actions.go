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

package cluster

import (
	"fmt"
	"sort"
	"sync"

	"github.com/pkg/errors"
)

// Action define a set of tasks to be executed on a `kind` cluster.
// Usage of actions allows to define repetitive, high level abstractions/workflows
// by composing lower level tasks
type Action interface {
	// Tasks returns the list of task that are identified by this action
	// Please note that the order of task is important, and it will be
	// respected during execution
	Tasks() []Task
}

// Task define a logical step of an action to be executed on a `kind` cluster.
// At exec time the logical step will then apply to the current cluster
// topology, and be planned for execution zero, one or many times accordingly.
type Task struct {
	// Description of the task
	Description string
	// TargetNodes define a function that identifies the nodes where this
	// task should be executed
	TargetNodes string
	// Run the func that implements the task action
	Run func(*KContext, *KNode, ActionFlags) error
}

// plannedTask defines a Task planned for execution on a given node.
type plannedTask struct {
	// task to be executed
	Task Task
	// node where the task should be executed
	Node *KNode

	// PlannedTask should respects the given order of actions and tasks
	actionIndex int
	taskIndex   int
}

// executionPlan contain an ordered list of Planned Tasks
// Please note that the planning order is critical for providing a
// predictable, "kubeadm friendly" and consistent execution order.
type executionPlan []*plannedTask

// internal registry of named Action implementations
var actionImpls = struct {
	impls map[string]func() Action
	sync.Mutex
}{
	impls: map[string]func() Action{},
}

// RegisterAction registers a new named actionBuilder function for use
func RegisterAction(name string, actionBuilderFunc func() Action) {
	actionImpls.Lock()
	actionImpls.impls[name] = actionBuilderFunc
	actionImpls.Unlock()
}

// getAction returns one instance of a registered action
func getAction(name string) (Action, error) {
	actionImpls.Lock()
	actionBuilderFunc, ok := actionImpls.impls[name]
	actionImpls.Unlock()
	if !ok {
		return nil, errors.Errorf("no Action implementation with name: %s", name)
	}
	return actionBuilderFunc(), nil
}

// KnownActions returns the list of know actions
func KnownActions() (actions []string) {
	actionImpls.Lock()
	for k := range actionImpls.impls {
		actions = append(actions, k)
	}
	actionImpls.Unlock()

	return actions
}

// newExecutionPlan creates an execution plan by applying logical step/task
// defined for each action to the actual cluster topology. As a result task
// could be executed zero, one or more times according with the target nodes
// selector defined for each task.
// The execution plan is ordered, providing a predictable, "kubeadm friendly"
// and consistent execution order; with this regard please note that the order
// of actions is important, and it will be respected by planning.
// TODO(fabrizio pandini): probably it will be necessary to add another criteria
//     for ordering planned task for the most complex workflows (e.g.
//     init-join-upgrade and then join again)
//     e.g. it should be something like "action group" where each action
//	   group is a list of actions
func newExecutionPlan(cfg *KContext, actions []string) (executionPlan, error) {
	// for each actionName
	var plan = executionPlan{}
	for i, name := range actions {
		// get the action implementation instance
		actionImpl, err := getAction(name)
		if err != nil {
			return nil, err
		}
		// for each logical tasks defined for the action
		for j, t := range actionImpl.Tasks() {
			// get the list of target nodes in the current topology
			targetNodes, _ := cfg.selectNodes(t.TargetNodes)
			for _, n := range targetNodes {
				// creates the planned task
				taskContext := &plannedTask{
					Node:        n,
					Task:        t,
					actionIndex: i,
					taskIndex:   j,
				}
				plan = append(plan, taskContext)
			}
		}
	}

	// sorts the list of planned task ensuring a predictable, "kubeadm friendly"
	// and consistent execution order
	sort.Sort(plan)
	return plan, nil
}

// Len of the executionPlan.
// It is required for making ExecutionPlan sortable.
func (t executionPlan) Len() int {
	return len(t)
}

// Less return the lower between two elements of the ExecutionPlan, where the
// lower element should be executed before the other.
// It is required for making ExecutionPlan sortable.
func (t executionPlan) Less(i, j int) bool {
	return t[i].ExecutionOrder() < t[j].ExecutionOrder()
}

// ExecutionOrder returns a string that can be used for sorting planned tasks
// into a predictable, "kubeadm friendly" and consistent order.
// NB. we are using a string to combine all the item considered into something
// that can be easily sorted using a lexicographical order
func (p *plannedTask) ExecutionOrder() string {
	return fmt.Sprintf("Node.ProvisioningOrder: %d - Node.Name: %s - actionIndex: %d - taskIndex: %d",
		// Then PlannedTask are grouped by machines, respecting the kubeadm node
		// ProvisioningOrder: first complete provisioning on bootstrap control
		// plane, then complete provisioning of secondary control planes, and
		// finally provision worker nodes.
		p.Node.ProvisioningOrder(),
		// Node name is considered in order to get a predictable/repeatable ordering
		// in case of many nodes with the same ProvisioningOrder
		p.Node.Name(),
		// If both the two criteria above are equal, the given order of actions will
		// be respected and, for each action, the predefined order of tasks
		// will be used
		p.actionIndex,
		p.taskIndex,
	)
}

// Swap two elements of the ExecutionPlan.
// It is required for making ExecutionPlan sortable.
func (t executionPlan) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
