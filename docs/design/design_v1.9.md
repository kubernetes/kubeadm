## Implementation design for kubeadm

<img src="https://raw.githubusercontent.com/cncf/artwork/master/projects/kubernetes/certified-kubernetes/1.9/color/certified-kubernetes-1.9-color.png" align="right" width="150px">`kubeadm init` and `kubeadm join` together provides a nice user experience for creating a best-practice but bare Kubernetes cluster from scratch.
However, it might not be obvious _how_ kubeadm does that.

This document strives to explain the phases of work that happen under the hood.
Also included is ComponentConfiguration API types for talking to kubeadm programmatically.

**Note:** Each and every one of the phases must be idempotent!

### The scope of kubeadm

The scope of `kubeadm init` and `kubeadm join` is to provide a smooth user experience for the user while bootstrapping a best-practice cluster.

The cluster that `kubeadm init` and `kubeadm join` set up should be:
 - Secure:
   - It should adopt latest best-practices like:
     - enforcing RBAC
     - using the Node Authorizer
     - using secure communication between the control plane components
     - using secure communication between the API server and the kubelets
     - making it possible to lock-down the kubelet API
     - locking down access to the API system components like the kube-proxy and kube-dns
     - locking down what a Bootstrap Token can access
 - Easy to use:
   - The user should not have to run anything more than a couple of commands, including:
     - `kubeadm init` on the master
     - `export KUBECONFIG=/etc/kubernetes/admin.conf`
     - `kubectl apply -f <network-of-choice.yaml>`
     - `kubeadm join --token <token> <master-ip>:<master-port>`
     - The `kubeadm join` request to add a node should be automatically approved
 - Extendable:
   - It should for example _not_ favor any network provider, instead configuring a network is out-of-scope
   - Should provide the possibility to use a config file for customizing various parameters

#### A note on constants / well-known values and paths

We have to draw the line somewhere about what should be configurable, what shouldn't, and what should be hard-coded in the binary.

We've decided to make the Kubernetes directory `/etc/kubernetes` a constant in the application, since it is clearly the given path in a majority of cases,
and the most intuitive location. Having that path configurable would confuse readers of an on-top-of-kubeadm-implemented deployment solution.

This means we aim to standardize:
 - `/etc/kubernetes/manifests` as the path where kubelet should look for static Pod manifests
   - Temporarily when bootstrapping, these manifests are present:
     - `etcd.yaml`
     - `kube-apiserver.yaml`
     - `kube-controller-manager.yaml`
     - `kube-scheduler.yaml`
 - `/etc/kubernetes/kubelet.conf` as the path where the kubelet should store its credentials to the API server.
 - `/etc/kubernetes/admin.conf` as the path from where the admin can fetch his/her superuser credentials.
 - Names of certificates files:
   - `ca.crt`, `ca.key` (CA certificate)
   - `apiserver.crt`, `apiserver.key` (API server certificate)
   - `apiserver-kubelet-client.crt`, `apiserver-kubelet-client.key` (client certificate for the apiservers to connect to the kubelets securely)
   - `sa.pub`, `sa.key` (a private key for signing ServiceAccount )
   - `front-proxy-ca.crt`, `front-proxy-ca.key` (CA for the front proxy)
   - `front-proxy-client.crt`, `front-proxy-client.key` (client cert for the front proxy client)
- Names of kubeconfig files:
  - `admin.conf`
  - `kubelet.conf` (`bootstrap-kubelet.conf` during TLS bootstrap)
  - `controller-manager.conf`
  - `scheduler.conf`


## `kubeadm init` phases

`kubeadm init` internal workflow consists of a sequence of atomic work tasks to perform.

The `kubeadm alpha phase` command allows users to invoke individually each task, and ultimately offers a reusable and composable API/toolbox that can be used by other Kubernetes bootstrap tools / by any IT automation tool / by advanced user for creating custom clusters.

### Preflight checks

`kubeadm` executes a set of preflight checks before starting the init, with the aim to verify preconditions and avoid common cluster startup problems.
In any case the user can skip specific preflight checks (or eventually all preflight checks) with the `--ignore-preflight-errors` option.

- [warning] If the Kubernetes version to use (passed with the `--kubernetes-version` flag) is at least one minor version higher than the kubeadm CLI version.
- Kubernetes system requirements:
  - if running on linux:
    - [error] if not Kernel 3.10+ or 4+ with specific KernelSpec.
    - [error] if required cgroups subsystem aren't in set up.
  - if using docker:
    - [error/warning] if Docker endpoint does not exist or does not work, if docker version >17.03.
    Note: starting from 1.9, kubeadm provides better support for CRI-generic functionality; in that case, docker specific controls are skipped or replaced by similar controls for crictl
- [error] if user is not root.
- [error] if hostname is not a valid DNS subdomain; [warning] if the host name cannot be reached via network lookup.
- [error] if kubelet version is lower that the minimum kubelet version supported by kubeadm (current minor -1).
- [error] if kubelet version is at least one minor higher than the required controlplane version (unsupported version skew).
- [warning] if kubelet service does not exist, if it is disabled.
- [warning] if firewalld is active.
- [error] if API.BindPort or ports 10250/10251/10252 are used.
- [Error] if /etc/kubernetes/manifest folder already exists and it is not empty.
- [Error] if /proc/sys/net/bridge/bridge-nf-call-iptables file does not exist/does not contain 1.
- [Error] if swap is on.
- [Error] if "ip", "iptables",  "mount", "nsenter" commands are not present in the command path.
- [warning] if "ebtables", "ethtool", "socat", "tc", "touch", "crictl" commands are not present in the command path.
- [warning] if extra arg flags for API server, controller manager,  scheduler contains some invalid options.
- [warning] if connection to https://API.AdvertiseAddress:API.BindPort goes thought proxy.
- [warning] if connection to services subnet goes thought proxy (only first address checked).
- [warning] if connection to Pods subnet goes thought proxy (only first address checked).
- If using docker:
  - [warning/error] if Docker service does not exist, if it is disabled, if it is not active.
- If using other cri engine:
  - [error] if crictl socket does not answer.
- If external etcd is provided:
  - [Error] if etcd version less than 3.0.14.
  - [Error] if certificates or keys are specified, but not provided.
- If external etcd is NOT provided:
  - [Error] if ports 2379 is used.
  - [Error] if Etcd.DataDir folder already exists and it is not empty.
- If authorization mode is ABAC, [Error] if abac_policy.json does not exist.
- If authorization mode is WebHook, [Error] if webhook_authz.conf does not exist.
- [Error] if advertise address is ipv6 and /proc/sys/net/bridge/bridge-nf-call-ip6tables does not exist/does not contain 1.

Please note that:

1. Preflight checks can be invoked individually with the `kubeadm alpha phase preflight` command.

### Generate the necessary certificates

`kubeadm` generates certificate and private key pairs for different purposes.
Certificates are stored by default in `/etc/kubernetes/pki`. This directory is configurable.

There should be:
 - A CA certificate (`ca.crt`) with its private key (`ca.key`).
 - An API server certificate (`apiserver.crt`) using `ca.crt` as the CA with its private key (`apiserver.key`). The certificate should:
   - Be a serving server certificate (`x509.ExtKeyUsageServerAuth`).
   - Contains altnames for:
     - The Kubernetes service's internal clusterIP (the first address in the services CIDR, e.g. `10.96.0.1` if service subnet is `10.96.0.0/12`).
     - Kubernetes DNS names (e.g.  `kubernetes.default.svc.cluster.local` if `--service-dns-domain` is `cluster.local`, `kubernetes.default.svc`, `kubernetes.default`, `kubernetes`).
     - The node-name.
     - The `--apiserver-advertise-address`.
     - Optional extra altnames specified by the user.
 - A client certificate for the API server to connect to the kubelets securely (`apiserver-kubelet-client.crt`) using `ca.crt` as the CA with its private key (`apiserver-kubelet-client.key`). The certificate should:
   - Be a client certificate (`x509.ExtKeyUsageClientAuth`).
   - Be in the `system:masters` organization.
 - A private key for signing ServiceAccount Tokens (`sa.key`) along with its public key (`sa.pub`).
 - A CA for the front proxy (`front-proxy-ca.crt`) with its key (`front-proxy-ca.key`).
 - A client cert for the front proxy client with its key  (`front-proxy-client.crt` and `front-proxy-client.key`) generate using `front-proxy-ca.crt` as the CA.


Please note that:

1. If a given certificate and private key pair both exist, and its content is evaluated compliant with the above specs, the existing files will be used and the generation phase for the given certificate skipped.
   This means the user can, for example, copy an existing CA to `/etc/kubernetes/pki/ca.{crt,key}` , and then  then `kubeadm` will use those files for signing the rest of the certs.
2. Only for the CA, it is possible to provide the `ca.crt` file but not the `ca.key` file, if all other certificates and kubeconfig files already are in place `kubeadm` recognize this condition and activates the ExternalCA , which also implies the `csrsigner`controller in controller-manager won't be started.
3. If `kubeadm` is running in "ExternalCA" mode; all the certificates must be provided as well, because  `kubeadm` cannot generate them by itself.
4. In case of `kubeadm`  executed in the `--dry-run` mode, certificates files are written in a temporary folder.
5. Certificate generation can be invoked individually with the `kubeadm alpha phase certs all` command.

### Generate kubeconfig files for the master components

There should be:
 - A kubeconfig file for kubeadm to use itself and the admin, `/etc/kubernetes/admin.conf`:
   - The "admin" here is defined as `kubeadm` itself and the actual person(s) that is administering the cluster and want to control the cluster:
     - With this file, the admin has full control (**root**) over the cluster.
   - Inside this file, a client certificate is generated from the `ca.crt` and `ca.key`. The client cert should:
     - Be a client certificate (`x509.ExtKeyUsageClientAuth`).
     - Be in the `system:masters` organization.
     - Include a CN, but that can be anything. `kubeadm` uses the `kubernetes-admin` CN.
 - A kubeconfig file for kubelet to use, `/etc/kubernetes/kubelet.conf`:
   - Inside this file, a client certificate is generated from the `ca.crt` and `ca.key`. The client cert should:
     - Be a client certificate (`x509.ExtKeyUsageClientAuth`).
     - Be in the `system:nodes` organization.
     - Have the CN `system:node:<hostname-lowercased>`.
 - A kubeconfig file for controller-manager, `/etc/kubernetes/controller-manager.conf`:
   - Inside this file, a client certificate is generated from the `ca.crt` and `ca.key`. The client cert should:
     - Be a client certificate (`x509.ExtKeyUsageClientAuth`).
     - Have the CN `system:kube-controller-manager`.
 - A kubeconfig file for scheduler, `/etc/kubernetes/scheduler.conf`:
   - Inside this file, a client certificate is generated from the `ca.crt` and `ca.key`. The client cert should:
     - Be a client certificate (`x509.ExtKeyUsageClientAuth`).
     - Have the CN `system:kube-scheduler`.

Please note that:

1. `ca.crt` is embedded in all the kubeconfig files.
2. If a given kubeconfig file exists, and its content is evaluated compliant with the above specs, the existing file will be used and the generation phase for the given kubeconfig skipped.
3. If `kubeadm` is running in ExternalCA mode, all the required kubeconfig must be provided by the user as well, because  `kubeadm` cannot generate any of them by itself.
4. In case of `kubeadm`  executed in the `--dry-run` mode, kubeconfig files are written in a temporary folder.
5. Kubeconfig files generation can be invoked individually with the `kubeadm alpha phase kubeconfig all` command.

### Generate static Pod manifests for control plane

Common properties for the control plane components:

- All static Pods are deployed on `kube-system` namespace.
- All static Pods get `tier:control-plane` and `component:{component-name}` labels.
- All static Pods get `scheduler.alpha.kubernetes.io/critical-pod` annotation. Note. this will be moved over to the proper solution of using Pod Priority and Preemption when ready.
- `hostNetwork: true` is set on all static Pods to allow control plane startup before a network is configured; accordingly:
  * The `address` that the controller-manager and the scheduler use to refer the API server is `127.0.0.1`.
  * If using a local etcd server, `etcd-servers` address  will be set to `127.0.0.1:2379`.
- Leader election is enabled for both the controller-manager and the scheduler.
- Controller-manager and the scheduler will reference kubeconfig files with their respective, unique identities.
- All static Pods get any extra flags specified by the user.
- All static Pods get any extra extra Volumes specified by the user (Host path).

Please note that:

1. All the images, for the  `--kubernetes-version`/current architecture, will be pulled from `gcr.io/google_containers`; In case an alternative image repository or CI image repository is specified this one will be used; In case a specific container image should be used for all control plane components, this one will be used.
2. In case of `kubeadm`  executed in the `--dry-run` mode, static Pods files are written in a temporary folder.
3. Static Pod manifest generation for master components can be invoked individually with the `kubeadm alpha phase controlplane all` command.

#### API server

The API server needs to know this in particular:
 - The `apiserver-advertise-address` and `apiserver-bind-port` to bind to (if not provided, those value defaults to the IP address of the default interface and port 6443).
 - The `service-cluster-ip-range` to use for services.
 - The `etcd-servers` address and related TLS settings `etcd-cafile`, `etcd-certfile`, `etcd-keyfile` if required; if an external etcd server won't be provided, a local etcd will be used (via host network).
 - If a cloud provider is specified, the corresponding `--cloud-provider` is configured, together with  the  `--cloud-config` path if such file exists. Note: this is experimental, alpha and will be removed in a future version
 - If kubeadm is invoked with `--feature-gates=HighAvailability`, the flag `--endpoint-reconciler-type=lease` is set, thus enabling automatic reconciliation of endpoints for the internal API server VIP.
 - If kubeadm is invoked with `--feature-gates=DynamicKubeletConfig`,  the corresponding feature on kube-apiserver is activated with `--feature-gates=DynamicKubeletConfig=true` flag.

Other flags that are set unconditionally:

 - `--insecure-port=0` to avoid insecure connections to the API server.
 - `--enable-bootstrap-token-auth=true` to enable the `BootstrapTokenAuthenticator` authentication module.
 - `--allow-privileged` to `true` (used e.g. by kube proxy).
 - `--client-ca-file` to `ca.crt`.
 - `--tls-cert-file` to `apiserver.crt`.
 - `--tls-private-key-file` to `apiserver.key`.
 - `--kubelet-client-certificate` to `apiserver-kubelet-client.crt`.
 - `--kubelet-client-key` to `apiserver-kubelet-client.key`.
 - `--service-account-key-file` to `sa.pub`.
 - `--requestheader-client-ca-file` to `front-proxy-ca.crt`.
 - `--admission-control` to:
    - [`Initializers`](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/) to enable [Dynamic Admission Control](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/).
    - [`NamespaceLifecycle`](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#namespacelifecycle) to avoid deletion of system reserved namespaces etc.
    - [`LimitRanger`](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#limitranger) and [`ResourceQuota`](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#resourcequota) to enforce limits on namespaces
    - [`ServiceAccount`](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#serviceaccount) to enforce service account automation.
    - [`PersistentVolumeLabel`](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#persistentvolumelabel)  attaches region or zone labels to PersistentVolumes as defined by the cloud provider. Note: This admission controller is deprecated and will be removed in a future version. It is not deployed by kubeadm by default with v1.9 onwards when not explicitly opting into using `gce` or `aws` as cloud providers..
    - [`DefaultStorageClass`](https://kubernetes.io/docs/admin/admission-controllers/#defaultstorageclass) to enforce default storage class on `PersistentVolumeClaim` objects.
    - [`DefaultTolerationSeconds`](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#defaultstorageclass) .
    - [`NodeRestriction`](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#noderestriction) to limit what a kubelet can modify (e.g. its own pods).
 - `--kubelet-preferred-address-types` to `InternalIP,ExternalIP,Hostname;` this makes `kubectl logs` and other apiserver -> kubelet communication work in environments where the hostnames of the nodes aren't resolvable.
 - `requestheader-client-ca-file` to`front-proxy-ca.crt`,  `proxy-client-cert-file` to `front-proxy-client.crt`,  `proxy-client-key-file` to `front-proxy-client.key` ,  and`--requestheader-username-headers=X-Remote-User`, `--requestheader-group-headers=X-Remote-Group`, `--requestheader-extra-headers-prefix=X-Remote-Extra-`, `--requestheader-allowed-names=front-proxy-client` so the front proxy ([API Aggregation](https://github.com/kubernetes/design-proposals-archive/blob/main/api-machinery/aggregated-api-servers.md)) communication is secure.

#### Controller manager

The controller-manager needs to know this in particular:
- If kubeadm is invoked specifying a `--pod-network-cidr`, the Subnet manager feature required for some CNI network plugins is enabled by
   setting `--allocate-node-cidrs=true`,  `--cluster-cidr` and `--node-cidr-mask-size` flags are set.
 - If a cloud provider is specified, the corresponding `--cloud-provider` is specified, together with  the  `--cloud-config` path if such configuration file exists. Note: this is experimental, alpha and will be removed in a future version

Other flags that are set unconditionally:
 - Enables all the default controllers plus `BootstrapSigner` and `TokenCleaner` controllers for TLS bootstrap.
 - `--root-ca-file` to `ca.crt`.
 - `--cluster-signing-cert-file` to `ca.crt`, if External CA mode is disabled, otherwise to `""`.
 - `--cluster-signing-key-file` to `ca.key`, if External CA mode is disabled, otherwise to `""`.
 - `--service-account-private-key-file` to `sa.key`.
 - `--use-service-account-credentials` to `true`.

#### Scheduler

Kubeadm doesn't set any special scheduler flags.

### Generate static Pod manifest for local etcd

If the user specified an external etcd this step will be skipped, otherwise a static manifest file will be generated for creating a local etcd instance running in a Pod with following attributes:

- listen on `localhost:2379` and use `HostNetwork=true`.
- make a `hostPath` mount out from the `dataDir` to the host's filesystem.
- Any extra flags specified by the user.

Please note that:

1. The etcd image will be pulled from `gcr.io/google_containers`. In case an alternative image repository is specified this one will be used; In case an alternative image name is specified, this one will be used.
2. in case of `kubeadm`  executed in the `--dry-run` mode, the etcd static Pod manifest is written in a temporary folder.
3. Static Pod manifest generation for local etcd can be invoked individually with the `kubeadm alpha phase etcd local` command.

### (optional and alpha in v1.9) Write init kubelet configuration

If kubeadm is invoked with `--feature-gates=DynamicKubeletConfig`, it writes the kubelet init configuration into `/var/lib/kubelet/config/init/kubelet` file.

The init configuration is used for starting the kubelet on this specific node, providing an alternative for the kubelet drop-in file; such configuration will be replaced by the kubelet base configuration as described in following steps.
See [online doc](https://kubernetes.io/docs/tasks/administer-cluster/kubelet-config-file/) for additional info.

Please note that:

1. to make dynamic kubelet configuration work, flag `--dynamic-config-dir=/var/lib/kubelet/config/dynamic` should be specified in `/etc/systemd/system/kubelet.service.d/10-kubeadm.conf`.
1. kubelet init configuration can be changed by using kubeadm MasterConfiguration file (`.kubeletConfiguration.baseConfig`).

### Wait for the control plane to come up

This is a critical moment in time for kubeadm clusters.
kubeadm waits until `localhost:6443/healthz` returns `ok`, however in order to detect deadlock conditions, kubeadm fails fast if `localhost:10255/healthz` (kubelet liveness) or `localhost:10255/healthz/syncloop` (kubelet readiness) don't return `ok`, respectively after 40 and 60 second.

kubeadm relies on the kubelet to pull the control plane images and run them properly as static Pods.
But there are (as we've seen) a lot of things that can go wrong. Most of them are network/resolv.conf/proxy related.

After the control plane is up, kubeadm completes a couple of tasks described in following paragraphs.

### (optional and alpha in v1.9) Write base kubelet configuration

If kubeadm is invoked with `--feature-gates=DynamicKubeletConfig`:

1. Write the kubelet base configuration into the `kubelet-base-config-v1.9`  ConfigMap in the `kube-system` namespace.
2. Creates RBAC rules for granting read access to that ConfigMap to all bootstrap tokens and all kubelets (`system:bootstrappers:kubeadm:default-node-token` and `system:nodes` groups).
3. Enable the dynamic kubelet configuration feature for the initial master node by pointing Node.spec.configSource to the newly-created configmap.

### Saves kubeadm MasterConfiguration in a ConfigMap for later reference

kubeadm saves the configuration passed to `kubeadm init`, either via flags or the config file,  in a ConfigMap named `kubeadm-config ` under `kube-system` namespace.

This will ensure that kubeadm actions executed in future (e.g `kubeadm upgrade`) will be able to determine the actual/current cluster state and make new decisions based on that data.

Please note that

1. Before uploading, sensitive information like e.g. the token are stripped from the configuration.
2. Upload of master configuration can be invoked individually with the `kubeadm alpha phase upload-config` command.
3. If you initialized your cluster using kubeadm v1.7.x or lower, you must create manually the master configuration ConfigMap before `kubeadm upgrade` to v1.8 . In order to facilitate this task, the `kubeadm config upload (from-flags|from-file)` was implemented.

### Mark master

As soon as the control plane is available, kubeadm executes following actions:

- Label  the master with `node-role.kubernetes.io/master=""`
- Taints  the master with `node-role.kubernetes.io/master:NoSchedule`

Please note that

1. Mark master phase can be invoked individually with the `kubeadm alpha phase mark-master` command.

### Configure TLS-Bootstrapping for node joining

Kubeadm uses [Authenticating with Bootstrap Tokens](https://kubernetes.io/docs/reference/access-authn-authz/bootstrap-tokens/) for joining new nodes to an existing cluster; for more details see also [design proposal](https://github.com/kubernetes/design-proposals-archive/blob/main/cluster-lifecycle/bootstrap-discovery.md).

`kubeadm init`  ensures that everything is properly configured for this process, and this includes following steps as well as setting API server and controller flags as already described in previous paragraphs.

Please note that

1. TLS bootstrapping for nodes can be configured with the `kubeadm alpha phase bootstrap-token all` command, executing configuration steps described in following paragraphs; alternatively, each step can be invoked individually.

#### Create a bootstrap token

`kubeadm init`  create a first bootstrap token, either generated automatically or provided by the user with the `--token` flag; as documented in bootstrap token specification, token should be saved as secrets with name `bootstrap-token-<token-id>` under `kube-system` namespace.

Please note that

1. The default token created by `kubeadm init` will be used to validate temporary user during TLS bootstrap process; those users will be member of  `system:bootstrappers:kubeadm:default-node-token` group.
2. The token has a limited validity, default 24 hours (the interval may be changed with the `—token-ttl` flag)
3. Additional tokens can be created with the `kubeadm token` command, that provide as well other useful functions for token management .

#### Allow joining nodes to call CSR API

kubeadm ensure that users in  `system:bootstrappers:kubeadm:default-node-token` group are able to access the certificate signing API.

This is implemented by creating a ClusterRoleBinding named `kubeadm:kubelet-bootstrap` between the  group above and the default RBAC role `system:node-bootstrapper`.

#### Setup auto approval for new bootstrap tokens

kubeadm ensures that the Bootstrap Token will get its CSR request automatically approved by the csrapprover controller.

This is implemented by creating ClusterRoleBinding named `kubeadm:node-autoapprove-bootstrap` between the  `system:bootstrappers:kubeadm:default-node-token` group and the default role `system:certificates.k8s.io:certificatesigningrequests:nodeclient`.

The role `system:certificates.k8s.io:certificatesigningrequests:nodeclient` should be created as well, granting POST permission to `/apis/certificates.k8s.io/certificatesigningrequests/nodeclient` (in v1.8  role will be automatically created by default).

#### Setup nodes certificate rotation with auto approval

kubeadm ensures that certificate rotation is enabled for nodes, and that new certificate request for nodes will get its CSR request automatically approved by the csrapprover controller.

This is implemented by creating ClusterRoleBinding named `kubeadm:node-autoapprove-certificate-rotation` between the  `system:nodes` group and the default role `system:certificates.k8s.io:certificatesigningrequests:selfnodeclient`.

#### Create the public `cluster-info` ConfigMap

This phase creates the `cluster-info` ConfigMap in the `kube-public` namespace.

Additionally it is created a role and a RoleBinding granting access for to the ConfigMap for unauthenticated users (i.e. users in RBAC group `system:unauthenticated`)

Please note that

1. The access to the `cluster-info` ConfigMap _is not_ rate-limited. This may or may not be a problem if you expose your master to the internet; worst-case scenario here is a DoS attack where an attacker uses all the in-flight requests the kube-apiserver can handle to serving the `cluster-info` ConfigMap.

### Install kube-proxy addon

A ServiceAccount for `kube-proxy` is created in the `kube-system` namespace; then kube-proxy is deployed as a DaemonSet:

- the credentials (`ca.crt` and `token`) to the master come from the ServiceAccount
- the location of the master comes from a ConfigMap
- the `kube-proxy` ServiceAccount is bound to the privileges in the `system:node-proxier` ClusterRole

Please note that

1. This phase can be invoked individually with the `kubeadm alpha phase addon kube-proxy`  command.

### Install dns addon

A ServiceAccount for `kube-dns` is created in the `kube-system` namespace.

Deploy the kube-dns Deployment and Service:

- it's the upstream kube-dns deployment relatively unmodified
- the `kube-dns` ServiceAccount is bound to the privileges in the `system:kube-dns` ClusterRole

Please note that

1. If kubeadm is invoked with `--feature-gates=CoreDNS`,  CoreDNS is installed instead of `kube-dns`.
2. This phase can be invoked individually with the `kubeadm alpha phase addon kube-dns`  command.

### (Optional and alpha in v1.9) self-hosting

This phase is performed only if `kubeadm init` is invoked with `—features-gates=self-hosting`

The self hosting phase basically replaces static Pods for control plane components with DaemonSets; this is achieved by executing following procedure for API server, scheduler and controller manager static Pods:

- Load the static Pod specification from disk
- Extract the PodSpec from that static Pod specification
- Mutate the PodSpec to be compatible with self-hosting, and more in detail:
  - add node selector attribute targeting nodes with`node-role.kubernetes.io/master=""`  label,
  - add a toleration for `node-role.kubernetes.io/master:NoSchedule` taint,
  - set `spec.DNSPolicy` to `ClusterFirstWithHostNet`
- Build a new DaemonSet object for the self-hosted component in question. Use the above mentioned PodSpec
- Create the DaemonSet resource in `kube-system` namespace. Wait until the Pods are running.
- Remove the static Pod manifest file. The kubelet will stop the original static Pod-hosted component that was running

Please note that:

1. Self hosting is not yet resilient to node restarts; this can be fixed with external checkpointing or with kubelet checkpointing for the control plane Pods.

2. If invoked with `—features-gates=StoreCertsInSecrets`  following additional steps will be executed
   - creation of `ca`,  `apiserver`,  `apiserver-kubelet-client`, `sa`, `front-proxy-ca`, `front-proxy-client`  TLS secrets in `kube-system` namespace with respective certificates and keys

     Note: Please note that storing the CA key in a Secret might have security implications.

   - creation of `scheduler.conf` and `controller-manager.conf` secrets in`kube-system` namespace with respective kubeconfig files

   - mutation of  all the Pod specs by replacing host path volumes with projected volumes from the secrets above

3. This phase can be invoked individually with the `kubeadm alpha phase selfhosting convert-from-staticpods`  command.

## `kubeadm join` phases

Similarly to `kubeadm init`, also `kubeadm join` internal workflow consists of a sequence of atomic work tasks to perform.

This is split into discovery (having the Node trust the Kubernetes Master) and TLS bootstrap (having the Kubernetes Master trust the Node).

see [Authenticating with Bootstrap Tokens](https://kubernetes.io/docs/reference/access-authn-authz/bootstrap-tokens/) , [design proposal](https://github.com/kubernetes/design-proposals-archive/blob/main/cluster-lifecycle/bootstrap-discovery.md).

### Preflight checks

`kubeadm` executes a set of preflight checks before starting the join, with the aim to verify preconditions and avoid common cluster startup problems.

Please note that:

1. `kubeadm join` preflight checks are basically a subset `kubeadm init` preflight checks
1. Starting from 1.9, kubeadm provides better support for CRI-generic functionality; in that case, docker specific controls are skipped or replaced by similar controls for crictl.
1. Starting from 1.9, kubeadm provides support for joining nodes running on Windows; in that case, linux specific controls are skipped.
1. In any case the user can skip specific preflight checks (or eventually all preflight checks) with the `--ignore-preflight-errors` option.

### Discovery cluster-info

There are 2 main schemes for discovery. The first is to use a shared token along with the IP address of the API server. The second is to provide a file (a subset of the standard kubeconfig file).

#### Shared token discovery

If `kubeadm join` is invoked with `--discovery-token`, token discovery is used; in this case the node basically retrieves the cluster CA certificates from the  `cluster-info` ConfigMap in the `kube-public` namespace.

In order to prevent "man in the middle" attacks, several steps are taken:

- First, the CA certificate is retrieved via insecure connection (note: this is possible because `kubeadm init` granted access to  `cluster-info` users for `system:unauthenticated` )
- Then the CA certificate goes through following validation steps:
  - "Basic validation", using the token ID against a JWT signature
  - "Pub key validation", using provided `--discovery-token-ca-cert-hash`. This value is available in the output of "kubeadm init" or can be calculated using standard tools (the hash is calculated over the bytes of the Subject Public Key Info (SPKI) object as in RFC7469). The `--discovery-token-ca-cert-hash flag` may be repeated multiple times to allow more than one public key.
  - as a additional validation, the CA certificate is retrieved via secure connection and then compared with the CA retrieved initially

Please note that:

1.  "Pub key validation" can be skipped passing `--discovery-token-unsafe-skip-ca-verification` flag; This weakens the kubeadm security model since others can potentially impersonate the Kubernetes Master.

#### File/https discovery

If `kubeadm join` is invoked with `--discovery-file`, file discovery is used; this file can be a local file or downloaded via an HTTPS URL; in case of HTTPS, the host installed CA bundle is used to verify the connection.

With file discovery, the cluster CA certificates is provided into the file itself; in fact, the discovery file is a kubeconfig file with only `server` and `certificate-authority-data` attributes set, e.g.:

```yaml
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: <really long certificate data>
    server: https://10.138.0.2:6443
  name: ""
contexts: []
current-context: ""
kind: Config
preferences: {}
users: []
```

Finally, when the connection with the cluster is established, kubeadm try to access the `cluster-info` ConfigMap, and if available, uses it.

## TLS Bootstrap

Once the cluster info are known, the file `bootstrap-kubelet.conf` is written, allowing kubelet to do TLS Bootstrapping (conversely in v.1.7 TLS were managed by kubeadm).

The TLS bootstrap mechanism uses the shared token to temporarily authenticate with the Kubernetes Master to submit a certificate signing request (CSR) for a locally created key pair.

The request is then automatically approved and the operation completes saving `ca.crt` file and `kubelet.conf` file to be used by kubelet for joining the cluster, while`bootstrap-kubelet.conf` is deleted.

Please note that:

- The temporary authentication is validated against the token saved during the `kubeadm init` process (or with additional tokens created with `kubeadm token`)
- The temporary authentication resolve to a user member of  `system:bootstrappers:kubeadm:default-node-token` group which was granted access to CSR api during the `kubeadm init` process
- The automatic CSR approval is managed by the csrapprover controller, according with configuration done the `kubeadm init` process

### (optional) Write init kubelet configuration

If kubeadm is invoked with `--feature-gates=DynamicKubeletConfig`:

1. Read the kubelet base configuration from the `kubelet-base-config-v1.9` ConfigMap in the `kube-system` namespace  using the Bootstrap Token credentials, and write it to disk as kubelet init configuration file  `/var/lib/kubelet/config/init/kubelet`.
2. As soon as kubelet starts with the Node's own credential (`/etc/kubernetes/kubelet.conf`), update current node configuration specifying that the source for the node/kubelet configuration is the above ConfigMap.

Please note that:

1. to make dynamic kubelet configuration work, flag `--dynamic-config-dir=/var/lib/kubelet/config/dynamic` should be specified in `/etc/systemd/system/kubelet.service.d/10-kubeadm.conf`.

## Extending `kubeadm`

There are two primary ways to extend `kubeadm`:
 - By setting CLI arguments or editing the lightweight `kubeadm init` API.
 - By running the phases you need separately and giving every phase the arguments it needs

The `kubeadm init` and `kubeadm join` APIs respectively are very limited in scope by design; That is where `kubeadm alpha phase` comes in, which gives you full power of the cluster creation.
