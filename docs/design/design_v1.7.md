## Implementation design for kubeadm 1.7

`kubeadm init` and `kubeadm join` together provides a nice user experience for creating a best-practice but bare Kubernetes cluster from scratch.
However, it might not be obvious _how_ kubeadm does that.

This document strives to explain the phases of work that happen under the hood.
Also included is ComponentConfiguration API types for talking to kubeadm programmatically.

**Note:** Each and every one of the phases must be idempotent!

### The scope of kubeadm

The scope of `kubeadm init` and `kubeadm join` is to provide a smooth user experience for the user while bootstrapping a best-practice cluster.

The cluster that `kubeadm init` and `kubeadm join` set up should be:
 - Secure
   - It should adopt latest best-practices like
     - enforcing RBAC
     - using the Node Authorizer
     - using secure communication between the control plane components
     - using secure communication between the API Server and the kubelets
     - making it possible to lock-down the kubelet API
     - locking down access to the API system components like the kube-proxy and kube-dns
     - locking down what a Bootstrap Token can access
 - Easy to use
   - The user should not have to run anything more than a couple of commands, including:
     - `kubeadm init` on the master
     - `export KUBECONFIG=/etc/kubernetes/admin.conf`
     - `kubectl apply -f <network-of-choice.yaml>`
     - `kubeadm join --token <token> <master>`
     - The `kubeadm join` request to add a node should be automatically approved
 - Extendable
   - It should for example _not_ favor any network provider, instead configuring a network is out-of-scope
   - Should provide a config file that can be used for customizing various parameters

#### A note on constants / well-known values and paths

We have to draw the line somewhere about what should be configurable, what shouldn't, and what should be hard-coded in the binary.

We've decided to make the Kubernetes directory `/etc/kubernetes` a constant in the application, since it is clearly the given path in a majority of cases,
and the most intuitive location. Having that path configurable would confuse readers of an on-top-of-kubeadm-implemented deployment solution.

This means we aim to standardize:
 - `/etc/kubernetes/manifests` as the path where kubelet should look for Static Pod manifests
   - Temporarily when bootstrapping, these manifests are present:
     - `etcd.yaml`
     - `kube-apiserver.yaml`
     - `kube-controller-manager.yaml`
     - `kube-scheduler.yaml`
 - `/etc/kubernetes/kubelet.conf` as the path where the kubelet should store its credentials to the API server.
 - `/etc/kubernetes/admin.conf` as the path from where the admin can fetch his/her superuser credentials.


## `kubeadm init` phases

### Phase 1: Generate the necessary certificates

`kubeadm` generates certificate and private key pairs for different purposes.
Certificates are stored by default in `/etc/kubernetes/pki`. This directory is configurable.

There should be:
 - a CA certificate (`ca.crt`) with its private key (`ca.key`)
 - an API Server certificate (`apiserver.crt`) using `ca.crt` as the CA with its private key (`apiserver.key`). The certificate should:
   - be a serving server certificate (`x509.ExtKeyUsageServerAuth`)
   - contain altnames for
     - the kubernetes' service's internal clusterIP and dns name (e.g. `10.96.0.1`, `kubernetes.default.svc.cluster.local`, `kubernetes.default.svc`, `kubernetes.default`, `kubernetes`)
     - the hostname of the node
       - **TODO:** I guess this might be a requested feature in opinionated setups, but might be a no-no in more advanced setups. Consensus here?
     - the IPv4 address of the default route
     - optional extra altnames that can be specified by the user
 - a client certificate for the apiservers to connect to the kubelets securely (`apiserver-kubelet-client.crt`) using `ca.crt` as the CA with its private key (`apiserver-kubelet-client.key`). The certificate should:
   - be a client certificate (`x509.ExtKeyUsageClientAuth`)
   - be in the `system:masters` organization
 - a private key for signing ServiceAccount Tokens (`sa.key`) along with its public key (`sa.pub`)
 - a CA for the front proxy (`front-proxy-ca.crt`) with its key (`front-proxy-ca.key`)
 - a client cert for the front proxy client (`front-proxy-client.crt`) using `front-proxy-ca.crt` as the CA with its key (`front-proxy-client.key`)


If a given certificate and private key pair both exist, the generation step will be skipped and those files will be validated and used for the prescribed use-case.
This means the user can, for example, prepopulate `/etc/kubernetes/pki/ca.{crt,key}` with an existing CA, which then will be used for signing the rest of the certs.

### Phase 2: Generate KubeConfig files for the master components

There should be:
 - a KubeConfig file for kubeadm to use itself and the admin: `/etc/kubernetes/admin.conf`
   - the "admin" here is defined as `kubeadm` itself and the actual person(s) that is administering the cluster and want to control the cluster
     - with this file, the admin has full control (**root**) over the cluster
   - inside this file, a client certificate is generated from the `ca.crt` and `ca.key`. The client cert should:
     - be a client certificate (`x509.ExtKeyUsageClientAuth`)
     - be in the `system:masters` organization
     - include a CN, but that can be anything. `kubeadm` uses the `kubernetes-admin` CN.
 - a KubeConfig file for kubelet to use: `/etc/kubernetes/kubelet.conf`
   - inside this file, a client certificate is generated from the `ca.crt` and `ca.key`. The client cert should:
     - be a client certificate (`x509.ExtKeyUsageClientAuth`)
     - be in the `system:nodes` organization
     - have the CN `system:node:<hostname-lowercased>`
 - a KubeConfig file for controller-manager: `/etc/kubernetes/controller-manager.conf`
   - inside this file, a client certificate is generated from the `ca.crt` and `ca.key`. The client cert should:
     - be a client certificate (`x509.ExtKeyUsageClientAuth`)
     - have the CN `system:kube-controller-manager`
 - a KubeConfig file for scheduler: `/etc/kubernetes/scheduler.conf`
   - inside this file, a client certificate is generated from the `ca.crt` and `ca.key`. The client cert should:
     - be a client certificate (`x509.ExtKeyUsageClientAuth`)
     - have the CN `system:kube-scheduler`

`ca.crt` is also embedded in all the KubeConfig files.

### Phase 3: Bootstrap the control plane by using Static Pods

#### etcd

Determine if the user specified external etcd options. If not, etcd should:
 - be spun up as a Static Pod
 - listen on `localhost:2379` and use `HostNetwork=true`
 - have `PodSpec.SecurityContext.SELinuxOptions.Type=spc_t` set because of https://github.com/kubernetes/kubeadm/issues/107
 - be at a minimum version `3.0.14`
 - make a `hostPath` mount out from the `dataDir` to the host's filesystem

#### API Server

The API Server needs to know this in particular:
 - The subnet to use for services
 - Where to find the etcd server
 - The address and port to bind to; defaults to the IP Address of the default interface and port 6443 for secure communication
 - Any extra flags and/or HostPath Volumes/VolumeMounts specified by the user

Other flags that are set:
 - The `BootstrapTokenAuthenticator` authentication module is enabled
 - `--client-ca-file` to `ca.crt`
 - `--tls-cert-file` to `apiserver.crt`
 - `--tls-private-key-file` to `apiserver.key`
 - `--kubelet-client-certificate` to `apiserver-kubelet-client.crt`
 - `--kubelet-client-key` to `apiserver-kubelet-client.key`
 - `--service-account-key-file` to `sa.pub`
 - `--requestheader-client-ca-file` to `front-proxy-ca.crt`
 - `--admission-control` to `NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota`
   - ...or whatever the recommended set of admission controllers is at a given version
 - `--storage-backend` to `etcd3`. Support for `etcd2` in kubeadm is dropped.
 - `--kubelet-preferred-address-types` to `InternalIP,ExternalIP,Hostname`
   - This makes `kubectl logs` and other apiserver -> kubelet communication work in environments where the hostnames of the nodes aren't resolvable
 - `--requestheader-username-headers=X-Remote-User`, `--requestheader-group-headers=X-Remote-Group`, `--requestheader-extra-headers-prefix=X-Remote-Extra-`, --requestheader-allowed-names=front-proxy-client` so the front proxy (API Aggregation) communication is secure.


#### Controller Manager

The controller-manager needs to know this in particular:
 - The Pod Network CIDR if any; also enables the Subnet Manager feature (required for some CNI network plugins)

Other flags that are set unconditionally:
 - The `BootstrapSigner` and `TokenCleaner` controllers are enabled
 - `--root-ca-file` to `ca.crt`
 - `--cluster-signing-cert-file` to `ca.crt`
 - `--cluster-signing-key-file` to `ca.key`
 - `--service-account-private-key-file` to `sa.key`
 - `--use-service-account-credentials` to `true`

#### Scheduler

kubeadm doesn't set any special scheduler flags.

Common properties for the control plane components:
 - Leader election is enabled for both the controller-manager and the scheduler
 - `HostNetwork: true` is present on all static pods since there is no network configured yet

#### Wait for the control plane to come up

This is a critical moment in time for kubeadm clusters.
kubeadm waits until `localhost:6443/healthz` returns `ok`

kubeadm relies on the kubelet to pull the control plane images and run them properly as Static Pods.
But there are (as we've seen) a lot of things that can go wrong. Most of them are network/resolv.conf/proxy related.

### Phase 4+: Post-bootstrap Phases

kubeadm completes a couple of tasks also after the control plane is up, namely these tasks:

#### Marks where the master is

This addon essentially does just this in pseudo-code:

```bash
kubectl taint node ${master_name} node-role.kubernetes.io/master:NoSchedule
kubectl label node ${master_name} node-role.kubernetes.io/master=""
```

#### cluster-info

This phase creates the `cluster-info` ConfigMap in the `kube-public` namespace as defined in [the Bootstrap Tokens proposal](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/cluster-lifecycle/bootstrap-discovery.md)
 - The `ca.crt` and the address/port of the apiserver is added to the `cluster-info` ConfigMap in the `kubeconfig` key
 - Exposes the `cluster-info` ConfigMap to unauthenticated users (i.e. users in RBAC group `system:unauthenticated`)

**Note:** The access to the `cluster-info` ConfigMap _is not_ rate-limited. 
This may or may not be a problem if you expose your master to the internet.
Worst-case scenario here is a DoS attack where an attacker uses all the in-flight requests the kube-apiserver can handle to serving the `cluster-info` ConfigMap.
TBD for v1.8

#### self-hosting

Parses the yaml static pod manifests in `/etc/kubernetes/manifests` and converts them to DaemonSets with master affinity:
 - TODO

#### kube-proxy

A ServiceAccount for `kube-proxy` is created in the `kube-system` namespace.

Deploy kube-proxy as a DaemonSet:
 - the credentials (`ca.crt` and `token`) to the master come from the ServiceAccount
 - the location of the master comes from a ConfigMap
 - the `kube-proxy` ServiceAccount is bound to the privileges in the `system:node-proxier` ClusterRole

#### kube-dns

A ServiceAccount for `kube-dns` is created in the `kube-system` namespace.

Deploy the kube-dns Deployment and Service:
 - it's the upstream kube-dns deployment relatively unmodified
 - the `kube-dns` ServiceAccount is bound to the privileges in the `system:kube-dns` ClusterRole

#### tls-bootstrap

The TLS Bootstrap ClusterRole (`system:node-bootstrapper`) is bound to the `system:bootstrappers` Group so Bootstrap Tokens are able to access the CSR API.

The `system:bootstrappers` Group is granted auto-approving status by it being able to `POST /apis/certificates.k8s.io/certificatesigningrequests/nodeclient`.
 - The auto-approving certificate controller in the controller-manager checks whether the poster of the CSR (in this case the Bootstrap Token) can POST to
   `/apis/certificates.k8s.io/certificatesigningrequests/nodeclient`. If the poster can, the controller approves the CSR.
 - This makes it possible to easily revoke the auto-approving functionality by removing the `ClusterRoleBinding` that grants Bootstrap Tokens that, or you can
   revoke access for all Bootstrap Tokens and instead make the auto-approving more granular by granting just a few users or tokens access to auto-approved credentials. 

## `kubeadm join` phases

### Phase 1: Fetch the `cluster-info` ConfigMap

This phase is skipped if
a) the `cluster-info` ConfigMap isn't exposed publicly (or created at all)
b) the user specified a file with the required `cluster-info` information

In the future, we can omit skipping the b) case and in the case valid information already is passed, refresh that information about the cluster location once again at
join time.

This phase basically issues a `GET /api/v1/namespaces/kube-public/configmaps/cluster-info` and validates the information there by using the token. You can find more
details about exactly how it does that in the Bootstrap Token discovery proposal.

## Phase 2: Do the TLS Bootstrap flow

`kubeadm` posts a CSR to the API server, which is then approved and signed by the certificates controller in the controller-manager.
`kubeadm` then writes the signed client certificate credentials to `/etc/kubernetes/kubelet.conf` for consumption by the kubelet.

**TODO:** It is planned that the kubelet will handle this phase 2 on its own in v1.8

## Extending `kubeadm`

There are a two primary ways to extend `kubeadm`:
 - By setting CLI arguments or editing the lightweight `kubeadm init` API.
 - By running the phases you need separately and giving every phase the arguments it needs

The `kubeadm init` and `kubeadm join` APIs respectively are very limited in scope.
They are there to make it possible to customize a couple of things to your needs, but doesn't allow for full flexibility.

### Open Questions

What do we have to change in this proposal/design doc to make kubeadm HA-friendly?
