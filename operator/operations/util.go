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

package operations

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
)

func createBasicTaskGroup(operation *operatorv1.Operation, taskdeploymentOrder string, taskdeploymentName string) operatorv1.RuntimeTaskGroup {
	gv := operatorv1.GroupVersion

	labels := map[string]string{}
	for k, v := range operation.Labels {
		labels[k] = v
	}
	labels[operatorv1.TaskGroupNameLabel] = taskdeploymentName
	labels[operatorv1.TaskGroupOrderLabel] = taskdeploymentOrder

	return operatorv1.RuntimeTaskGroup{
		TypeMeta: metav1.TypeMeta{
			Kind:       gv.WithKind("TaskGroup").Kind,
			APIVersion: gv.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            fmt.Sprintf("%s-%s-%s", operation.Name, taskdeploymentOrder, taskdeploymentName), //TODO: GeneratedName?
			Labels:          labels,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(operation, operation.GroupVersionKind())},
		},
		Spec: operatorv1.RuntimeTaskGroupSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: operatorv1.RuntimeTaskTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:            labels,
					CreationTimestamp: metav1.Now(),
				},
				Spec: operatorv1.RuntimeTaskSpec{
					Commands: []operatorv1.CommandDescriptor{},
				},
			},
		},
		Status: operatorv1.RuntimeTaskGroupStatus{
			Phase: string(operatorv1.OperationPhasePending),
		},
	}
}

func setCPSelector(t *operatorv1.RuntimeTaskGroup) {
	t.Spec.NodeSelector = metav1.LabelSelector{
		MatchLabels: map[string]string{
			"node-role.kubernetes.io/master": "",
		},
	}
}

func setCP1Selector(t *operatorv1.RuntimeTaskGroup) {
	t.Spec.NodeSelector = metav1.LabelSelector{
		MatchLabels: map[string]string{
			"node-role.kubernetes.io/master": "",
		},
	}
	t.Spec.NodeFilter = string(operatorv1.RuntimeTaskGroupNodeFilterHead)
}

func setCPNSelector(t *operatorv1.RuntimeTaskGroup) {
	t.Spec.NodeSelector = metav1.LabelSelector{
		MatchLabels: map[string]string{
			"node-role.kubernetes.io/master": "",
		},
	}
	t.Spec.NodeFilter = string(operatorv1.RuntimeTaskGroupNodeFilterTail)
}

func setWSelector(t *operatorv1.RuntimeTaskGroup) {
	t.Spec.NodeSelector = metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      "node-role.kubernetes.io/master",
				Operator: metav1.LabelSelectorOpDoesNotExist,
			},
		},
	}
}

func fixupCustomTaskGroup(operation *operatorv1.Operation, taskgroup operatorv1.RuntimeTaskGroup, taskdeploymentOrder string) operatorv1.RuntimeTaskGroup {
	gv := operatorv1.GroupVersion

	//TODO: consider if to preserve labels from custom taskgroup, taskgroup.Spec.Selector, taskgroup.Spec.Template

	labels := map[string]string{}
	for k, v := range operation.Labels {
		labels[k] = v
	}
	labels[operatorv1.TaskGroupNameLabel] = taskgroup.Name
	labels[operatorv1.TaskGroupOrderLabel] = taskdeploymentOrder

	taskgroup.SetGroupVersionKind(gv.WithKind("TaskGroup"))
	taskgroup.SetLabels(labels)
	taskgroup.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(operation, operation.GroupVersionKind())})
	taskgroup.Spec.Selector.MatchLabels = labels
	taskgroup.Spec.Template.SetLabels(labels)
	taskgroup.Spec.Template.SetCreationTimestamp(metav1.Now())
	taskgroup.Status = operatorv1.RuntimeTaskGroupStatus{
		Phase: string(operatorv1.OperationPhasePending),
	}

	return taskgroup
}
