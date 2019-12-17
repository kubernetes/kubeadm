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
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
	"k8s.io/kubeadm/operator/operations"
)

func getImage(c client.Client, namespace, name string) (string, error) {
	pod := &corev1.Pod{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
	err := c.Get(
		context.Background(), key, pod,
	)
	if err != nil {
		return "", errors.Errorf("error reading pod %s/%s", namespace, name)
	}

	var managerImage string
	for _, c := range pod.Spec.Containers {
		if c.Name == "manager" {
			managerImage = c.Image
		}
	}

	if managerImage == "" {
		return "", errors.Errorf("unable to get Image for manager container in %s/%s", namespace, name)
	}

	return managerImage, nil
}

func getDaemonSet(c client.Client, operation *operatorv1.Operation, namespace string) (*appsv1.DaemonSet, error) {
	daemonSet := &appsv1.DaemonSet{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      daemonSetName(operation.Name),
	}
	err := c.Get(
		context.Background(), key, daemonSet,
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return daemonSet, nil
}

func daemonSetName(operationName string) string {
	return fmt.Sprintf("controller-agent-%s", operationName)
}

func hostPathTypePtr(t corev1.HostPathType) *corev1.HostPathType {
	return &t
}

func createDaemonSet(c client.Client, operation *operatorv1.Operation, namespace, image string, metricsRBAC bool) error {
	labels := map[string]string{}
	for k, v := range operation.Labels {
		labels[k] = v
	}

	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       namespace,
			Name:            daemonSetName(operation.Name),
			Labels:          labels,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(operation, operation.GroupVersionKind())},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:            labels,
					CreationTimestamp: metav1.Now(),
				},
				Spec: corev1.PodSpec{
					Tolerations: []corev1.Toleration{
						{
							Key:    "node-role.kubernetes.io/master",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "agent",
							Image: image,
							Command: []string{
								"/manager",
							},
							Args: []string{
								"--mode=agent",
								"--agent-node-name=$(MY_NODE_NAME)",
								fmt.Sprintf("--agent-operation=%s", operation.Name),
							},
							Env: []corev1.EnvVar{
								{
									Name: "MY_NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("30Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("20Mi"),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: pointer.BoolPtr(true),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kubeadm-binary",
									MountPath: "/usr/bin/kubeadm",
								},
								{
									Name:      "etc-kubernetes",
									MountPath: "/etc/kubernetes",
								},
							},
						},
					},
					TerminationGracePeriodSeconds: pointer.Int64Ptr(10),
					HostNetwork:                   true,
					Volumes: []corev1.Volume{
						{
							Name: "kubeadm-binary",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/usr/bin/kubeadm",
									Type: hostPathTypePtr(corev1.HostPathFile),
								},
							},
						},
						{
							Name: "etc-kubernetes",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/kubernetes",
									Type: hostPathTypePtr(corev1.HostPathDirectory),
								},
							},
						},
					},
				},
			},
		},
	}

	extraLabels, err := operations.DaemonSetNodeSelectorLabels(operation)
	if err != nil {
		return errors.Wrapf(err, "failed to get NodeSelector for the operation DaemonSet %s/%s", daemonSet.Namespace, daemonSet.Name)
	}
	if len(extraLabels) > 0 {
		daemonSet.Spec.Template.Spec.NodeSelector = extraLabels
	}

	if metricsRBAC {
		// Force /metrics to be accessible only locally
		daemonSet.Spec.Template.Spec.Containers[0].Args = append(daemonSet.Spec.Template.Spec.Containers[0].Args,
			"--metrics-addr=127.0.0.1:8080",
		)

		// Expose /metrics via rbac-proxy sidecar
		daemonSet.Spec.Template.Spec.Containers = append(daemonSet.Spec.Template.Spec.Containers,
			corev1.Container{
				Name:  "kube-rbac-proxy",
				Image: "gcr.io/kubebuilder/kube-rbac-proxy:v0.4.0",
				Args: []string{
					"--secure-listen-address=0.0.0.0:8443",
					"--upstream=http://127.0.0.1:8080/",
					"--logtostderr=true",
					"--v=10",
				},
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: 8443,
						Name:          "https",
					},
				},
			},
		)

	} else {
		// Expose /metrics on default (insecure) port
		if daemonSet.Annotations == nil {
			daemonSet.Annotations = map[string]string{}
		}
		daemonSet.Annotations["prometheus.io/scrape"] = "true"
		daemonSet.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
			{
				ContainerPort: 8080,
				Name:          "metrics",
				Protocol:      corev1.ProtocolTCP,
			},
		}
	}

	if err := c.Create(
		context.Background(), daemonSet,
	); err != nil {
		return errors.Wrapf(err, "failed to create DaemonSet %s/%s", daemonSet.Namespace, daemonSet.Name)
	}

	return nil
}

func deleteDaemonSet(c client.Client, daemonSet *appsv1.DaemonSet) error {
	if err := c.Delete(
		context.Background(), daemonSet,
	); err != nil {
		return errors.Wrapf(err, "failed to delete DaemonSet %s/%s", daemonSet.Namespace, daemonSet.Name)
	}

	return nil
}

func listTaskGroupsByLabels(c client.Client, labels map[string]string) (*operatorv1.RuntimeTaskGroupList, error) {
	taskdeployments := &operatorv1.RuntimeTaskGroupList{}
	if err := c.List(
		context.Background(), taskdeployments,
		client.MatchingLabels(labels),
	); err != nil {
		return nil, err
	}

	return taskdeployments, nil
}

func recordPausedChange(recorder record.EventRecorder, obj runtime.Object, current, new bool, args ...string) {
	if current != new {
		reasonVerb := "Paused"
		messageAction := "set to pause"
		if !new {
			reasonVerb = "Restarted"
			messageAction = "set for restart"
		}

		reason := fmt.Sprintf("%s%s", obj.GetObjectKind().GroupVersionKind().Kind, reasonVerb)
		message := fmt.Sprintf("%s %s", obj.GetObjectKind().GroupVersionKind().Kind, messageAction)
		if len(args) > 0 {
			message = fmt.Sprintf("%s %s", message, strings.Join(args, " "))
		}
		recorder.Event(obj, corev1.EventTypeNormal, reason, message)
	}
}

func operationToTaskGroupRequests(c client.Client, o handler.MapObject) []ctrl.Request {
	var result []ctrl.Request

	operation, ok := o.Object.(*operatorv1.Operation)
	if !ok {
		return nil
	}

	actual, err := listTaskGroupsByLabels(c, operation.Labels)
	if err != nil {
		return nil
	}

	for _, ms := range actual.Items {
		name := client.ObjectKey{Namespace: ms.Namespace, Name: ms.Name}
		result = append(result, ctrl.Request{NamespacedName: name})
	}

	return result
}

func getOwnerOperation(ctx context.Context, c client.Client, obj metav1.ObjectMeta) (*operatorv1.Operation, error) {
	for _, ref := range obj.OwnerReferences {
		if ref.Kind == "Operation" && ref.APIVersion == operatorv1.GroupVersion.String() {
			operation := &operatorv1.Operation{}
			key := client.ObjectKey{
				Namespace: obj.Namespace,
				Name:      ref.Name,
			}
			if err := c.Get(ctx, key, operation); err != nil {
				return nil, errors.Wrapf(err, "error reading controller ref for %s/%s", obj.Namespace, obj.Name)
			}
			return operation, nil
		}
	}
	return nil, errors.Errorf("missing controller ref for %s/%s", obj.Namespace, obj.Name)
}

type matchingSelector struct {
	selector labels.Selector
}

func (m matchingSelector) ApplyToList(opts *client.ListOptions) {
	opts.LabelSelector = m.selector
}

func listNodesBySelector(c client.Client, selector *metav1.LabelSelector) (*corev1.NodeList, error) {
	s, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert TaskGroup.Spec.NodeSelector to a selector")
	}

	o := matchingSelector{selector: s}

	nodes := &corev1.NodeList{}
	if err := c.List(
		context.Background(), nodes,
		o,
	); err != nil {
		return nil, err
	}

	return nodes, nil
}

func filterNodes(nodes *corev1.NodeList, filter operatorv1.RuntimeTaskGroupNodeFilter) []corev1.Node {
	if len(nodes.Items) == 0 {
		return nodes.Items
	}

	if filter == operatorv1.RuntimeTaskGroupNodeFilterAll || filter == operatorv1.RuntimeTaskGroupNodeUnknownFilter {
		return nodes.Items
	}

	// in order to ensure a predictable result, nodes are sorted by name before applying the filter
	sort.Slice(nodes.Items, func(i, j int) bool { return nodes.Items[i].Name < nodes.Items[j].Name })

	if filter == operatorv1.RuntimeTaskGroupNodeFilterHead {
		return nodes.Items[:1]
	}

	// filter == operatorv1alpha1.TaskGroupNodeFilterTail
	return nodes.Items[1:]
}

func listTasksBySelector(c client.Client, selector *metav1.LabelSelector) (*operatorv1.RuntimeTaskList, error) {
	selectorMap, err := metav1.LabelSelectorAsMap(selector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert TaskGroup.Spec.Selector to a selector")
	}

	tasks := &operatorv1.RuntimeTaskList{}
	if err := c.List(
		context.Background(), tasks,
		client.MatchingLabels(selectorMap),
	); err != nil {
		return nil, err
	}

	return tasks, nil
}

func taskGroupToTaskRequests(c client.Client, o handler.MapObject) []ctrl.Request {
	var result []ctrl.Request

	taskgroup, ok := o.Object.(*operatorv1.RuntimeTaskGroup)
	if !ok {
		return nil
	}

	actual, err := listTasksBySelector(c, &taskgroup.Spec.Selector)
	if err != nil {
		return nil
	}

	for _, ms := range actual.Items {
		name := client.ObjectKey{Namespace: ms.Namespace, Name: ms.Name}
		result = append(result, ctrl.Request{NamespacedName: name})
	}

	return result
}

func getOwnerTaskGroup(ctx context.Context, c client.Client, obj metav1.ObjectMeta) (*operatorv1.RuntimeTaskGroup, error) {
	for _, ref := range obj.OwnerReferences {
		if ref.Kind == "RuntimeTaskGroup" && ref.APIVersion == operatorv1.GroupVersion.String() {
			taskgroup := &operatorv1.RuntimeTaskGroup{}
			key := client.ObjectKey{
				Namespace: obj.Namespace,
				Name:      ref.Name,
			}
			if err := c.Get(ctx, key, taskgroup); err != nil {
				return nil, errors.Wrapf(err, "error reading controller ref for %s/%s", obj.Namespace, obj.Name)
			}
			return taskgroup, nil
		}
	}
	return nil, errors.Errorf("missing controller ref for %s/%s", obj.Namespace, obj.Name)
}
