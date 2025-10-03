/*
Copyright 2021 The Kubernetes Authors.

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

package assets

// KindnetImage185 is a image for kindnet
const KindnetImage185 = "ghcr.io/aojea/kindnetd:v1.8.5"

// KindnetManifest185 holds a kindnet manifest
const KindnetManifest185 = `
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: kindnet
rules:
  - apiGroups:
      - ""
    resources:
      - nodes
    verbs:
      - list
      - watch
      - patch
  - apiGroups:
      - ""
    resources:
      - nodes/proxy
      - nodes/configz
    verbs:
      - get
  - apiGroups:
     - ""
    resources:
      - configmaps
    verbs:
      - get
  - apiGroups:
      - ""
    resources:
      - pods
      - namespaces
    verbs:
      - list
      - watch
  - apiGroups:
     - "networking.k8s.io"
    resources:
      - networkpolicies
    verbs:
      - list
      - watch
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: kindnet
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kindnet
subjects:
- kind: ServiceAccount
  name: kindnet
  namespace: kube-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kindnet
  namespace: kube-system
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: kindnet
  namespace: kube-system
  labels:
    tier: node
    app: kindnet
    k8s-app: kindnet
spec:
  selector:
    matchLabels:
      app: kindnet
  template:
    metadata:
      labels:
        tier: node
        app: kindnet
        k8s-app: kindnet
    spec:
      hostNetwork: true
      tolerations:
      - operator: Exists
        effect: NoSchedule
      serviceAccountName: kindnet
      initContainers:
      - name: install-cni-bin
        image: ghcr.io/aojea/kindnetd:v1.8.5
        command: ['sh', '-c', 'cat /opt/cni/bin/cni-kindnet > /cni/cni-kindnet ; chmod +x /cni/cni-kindnet']
        volumeMounts:
        - name: cni-bin
          mountPath: /cni
      containers:
      - name: kindnet-cni
        image: ghcr.io/aojea/kindnetd:v1.8.5
        command:
        - /bin/kindnetd
        - --hostname-override=$(NODE_NAME)
        - --network-policy=true
        - --admin-network-policy=false
        - --baseline-admin-network-policy=false
        - --masquerading=true
        - --dns-caching=true
        - --disable-cni=false
        - --fastpath-threshold=20
        - --ipsec-overlay=false
        - --nat64=true
        - --v=2
        env:
        - name: HOST_IP
          valueFrom:
            fieldRef:
              fieldPath: status.hostIP
        - name: POD_IP
          valueFrom:
            fieldRef:
              fieldPath: status.podIP
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        volumeMounts:
        - name: cni-cfg
          mountPath: /etc/cni/net.d
        - name: var-lib-kindnet
          mountPath: /var/lib/cni-kindnet
        resources:
          requests:
            cpu: "100m"
            memory: "50Mi"
        securityContext:
          privileged: true
      volumes:
      - name: cni-bin
        hostPath:
          path: /opt/cni/bin
          type: DirectoryOrCreate
      - name: cni-cfg
        hostPath:
          path: /etc/cni/net.d
          type: DirectoryOrCreate
      - name: var-lib-kindnet
        hostPath:
          path: /var/lib/cni-kindnet
          type: DirectoryOrCreate
---
`
