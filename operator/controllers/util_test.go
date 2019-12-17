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

	"k8s.io/kubeadm/operator/api/v1alpha1"
	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
)

//TODO: more tests

func Test_filterNodes(t *testing.T) {
	nodes := &corev1.NodeList{
		Items: []corev1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "n3",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "n1",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "n2",
				},
			},
		},
	}

	type args struct {
		filter v1alpha1.RuntimeTaskGroupNodeFilter
	}
	tests := []struct {
		name string
		args args
		want []corev1.Node
	}{
		{
			name: "filter all return all nodes",
			args: args{
				filter: operatorv1.RuntimeTaskGroupNodeFilterAll,
			},
			want: nodes.Items,
		},
		{
			name: "Filter head return the first node",
			args: args{
				filter: operatorv1.RuntimeTaskGroupNodeFilterHead,
			},
			want: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "n1",
					},
				},
			},
		},
		{
			name: "Filter tail return the last two nodes",
			args: args{
				filter: operatorv1.RuntimeTaskGroupNodeFilterTail,
			},
			want: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "n2",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "n3",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterNodes(nodes, tt.args.filter); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterNodes() = %v, want %v", got, tt.want)
			}
		})
	}
}
