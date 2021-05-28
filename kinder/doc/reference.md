# Kinder

## Create a test cluster

You can create a cluster in kinder using `kinder create cluster`, that is a wrapper on the `kind create cluster` command.

However, additional flags are implemented for enabling following use cases:

### Create *only* nodes

By default kinder stops the cluster creation process before executing `kubeadm init` and `kubeadm join`;
This will give you nodes ready for installing Kubernetes and more specifically

- the necessary prerequisites already installed on all nodes
- in case of more than one control-plane node exists in the cluster, a pre-configured external load balancer

### Testing different cluster topologies

You can use the `--control-plane-nodes <num>` flag and/or the `--worker-nodes <num>`  flag
as a shortcut for creating different cluster topologies. e.g.

```bash
# create a cluster with two worker nodes
kinder create cluster --worker-nodes=2

# create a cluster with two control-pane nodes
kinder create cluster ---control-plane-nodes=2
```

Please note that a load balancer node will be automatically create when there are more than
one control-plane node; if necessary, you can use `--external-load-balancer` flag to explicitly
request the creation of an external load balancer node.

It is also possible to create an external etcd cluster using the `--external-etcd` flag.

More sophisticated cluster topologies can be achieved using the kind config file, like e.g. customizing
kubeadm-config or specifying volume mounts. see [kind documentation](https://kind.sigs.k8s.io/docs/user/quick-start/#configuring-your-kind-cluster)
for more details.

## Working on nodes

You can use `docker exec` and `docker cp`  to work on nodes.

```bash
# check the content of the /kind/kubeadm.conf file
docker exec kind-control-plane cat kind/kubeadm.conf

# execute a command on the kind-control-plane container (the control-plane node)
docker exec kind-worker1 \
      kubeadm init --config=/kind/kubeadm.conf --ignore-preflight-errors=all

# override the kubeadm binary on the on the kind-control-plane container
# with a locally built kubeadm binary
docker cp \
      $working_dir/kubernetes/bazel-bin/cmd/kubeadm/linux_amd64_pure_stripped/kubeadm \
      kind-control-plane:/usr/bin/kubeadm
```

On top of that, kinder offers you three commands for helping you working on nodes:

- `kinder do` allowing you to execute actions (repetitive tasks/sequence of commands) on nodes
- `kinder exec`,  a topology aware wrapper on docker `docker exec`
- `kinder cp`, a topology aware wrapper on docker `docker cp`

### kinder do

`kinder do` is the kinder swiss knife.

It allows to execute actions (repetitive tasks/sequence of commands) on one or more nodes
in the local Kubernetes cluster.

```bash
# Execute kubeadm init, installs the CNI plugin and copy the kubeconfig file on the host
kinder do kubeadm-init
```

All the actions implemented in kinder are by design "developer friendly", in the sense that
all the command output will be echoed and all the step will be documented.
Following actions are available:

| action          | Notes                                                        |
| --------------- | ------------------------------------------------------------ |
| kubeadm-config  | Creates `/kind/kubeadm.conf` files on nodes (this action is automatically executed during `kubeadm-init` or `kubeadm-join`). Available options are:<br />`--copy-certs=auto` instruct kubeadm to prepare for use the automatic copy cert feature. <br />`--discover-mode` instruct kubeadm to use a specific discovery mode when doing kubeadm join.<br /> `--only-node` to execute this action only on a specific node. <br /> `--dry-run`|
| loadbalancer    | Update the load balancer configuration, if present (this action is automatically executed during `kubeadm-init` or `kubeadm-join`) .|
| kubeadm-init    | Executes the kubeadm-init workflow, installs the CNI plugin and then copies the kubeconfig file on the host machine. Available options are:<br /> `--use-phases` triggers execution of the init workflow by invoking single phases.<br />`--copy-certs=auto` instruct kubeadm to use the automatic copy cert feature.<br /> `--dry-run`||
| manual-copy-certs      | Implement the manual copy of certificates to be shared across control-plane nodes (n.b. manual means not managed by kubeadm) Available options are:<br />  `--only-node` to execute this action only on a specific node. <br /> `--dry-run`||
| kubeadm-join    | Executes the kubeadm-join workflow both on secondary control plane nodes and on worker nodes. Available options are:<br /> `--use-phases` triggers execution of the init workflow by invoking single phases.<br />`--copy-certs=auto` instruct kubeadm to use the automatic copy cert feature.<br />`--discover-mode` instruct kubeadm to use a specific discovery mode when doing kubeadm join.<br /> `--only-node` to execute this action only on a specific node. <br /> `--dry-run`||
| kubeadm-upgrade |Executes the kubeadm upgrade workflow and upgrading K8s. Available options are:<br /> `--upgrade-version` for defining the target K8s version.<br />`--only-node` to execute this action only on a specific node.                           <br /> `--dry-run`|
| kubeadm-reset   | Executes the kubeadm-reset workflow on all the nodes. Available options are:<br />  `--only-node` to execute this action only on a specific node. Available options are:<br /> `--dry-run`||
| cluster-info    | Returns a summary of cluster info including<br />- List of nodes<br />- list of pods<br />- list of images used by pods<br />- list of etcd members |
| smoke-test      | Implements a non-exhaustive set of tests that aim at ensuring that the most important functions of a Kubernetes cluster work |
| setup-external-ca  | Setups the cluster for external CA mode:<br />- Generates shared certificates and kubeconfig files on the bootstrap node and copies them to other CP nodes<br />- Copies the CA to all nodes and signs kubelet.conf files required for bootstrap<br />- Deletes the ca.key from all nodes

### kinder exec

`kinder exec` provide a topology aware wrapper on docker `docker exec` .

```bash
# check the kubeadm version on all the nodes
kinder exec @all -- kubeadm version

# run kubeadm join on all the worker nodes
kinder exec @w* -- kubeadm join 172.17.0.2:6443 --token abcdef.0123456789abcdef ...

# run kubectl command inside the bootstrap control-plane node
kinder exec @cp1 -- kubectl --kubeconfig=/etc/kubernetes/admin.conf cluster-info
```

Following node selectors are available

| selector | return the following nodes                                   |
| -------- | ------------------------------------------------------------ |
| @all     | all the Kubernetes nodes in the cluster.<br />(control-plane and worker nodes are included, load balancer and etcd not) |
| @cp*     | all the control-plane nodes                                  |
| @cp1     | the bootstrap-control plane node                             |
| @cpN     | the secondary control plane nodes                            |
| @w*      | all the worker nodes                                         |
| @lb      | the external load balancer                                   |
| @etcd    | the external etcd                                            |

As alternative to node selector, the node name (the container name without the cluster name prefix) can be used to target actions to a specific node.

```bash
# run kubeadm join on the first worker node only
kinder exec worker1 -- kubeadm join 172.17.0.2:6443 --token abcdef.0123456789abcdef ...
```

### kinder cp

`kinder cp` provide a topology aware wrapper on docker `docker cp` . Following feature are supported:

```bash
# copy to the host the /kind/kubeadm.conf file existing on the bootstrap control-plane node
kinder cp @cp1:kind/kubeadm.conf kubeadm.conf

# copy to the bootstrap control-plane node a local kubeadm.conf file
kinder cp kubeadm.conf @cp1:kind/kubeadm.conf

# override the kubeadm binary on all the nodes with a locally built kubeadm binary
kinder cp \
      $working_dir/kubernetes/bazel-bin/cmd/kubeadm/linux_amd64_pure_stripped/kubeadm \
      @all:/usr/bin/kubeadm
```

> Please note that,  `docker cp` or `kinder cp`  allows you to replace the kubeadm binary on existing nodes. If you want to replace the kubeadm binary on nodes that you create in future, please check altering node images paragraph

## Altering images

Kind can be extremely efficient when the node image contains all the necessary artifacts.

kinder allows kubeadm contributor to exploit this feature by implementing the `kinder build node-image-variant` command, that takes a node-image (or a bare base image) and allows to build variants by:

- Adding a Kubernetes version to be used for initializing the cluster (as an alternative to `build node-image` already supported by kind)
- Adding new pre-loaded images that will be made available on all nodes at cluster creation time
- Replacing the kubeadm binary installed in the cluster, e.g. with a locally build version of kubeadm
- Replacing the kubelet binary installed in the cluster, e.g. with a locally build version of kubelet
- Adding binaries for a second Kubernetes version to be used for upgrade testing

`kinder build node-image-variant` can read artifacts to be added to the base image from following sources

- a version, e.g. v1.14.0 or v1.15.0-alpha.0.100+78573805a7292a
- a release build label, e.g. release/stable, release/stable-1.13, release/latest-14
- a ci build label, e.g. ci/latest, ci/latest-1.14
- a remote repository, e.g. <http://k8s.mycompany.com/>
- a local folder, as shown in the examples above.

### Add init packages

```bash
kinder build node-image-variant \
     --base-image kindest/base:latest \
     --image kindest/node:PR12345 \
     --with-init-artifacts v1.14.0

kinder build node-image-variant \
     --base-image kindest/base:latest \
     --image kindest/node:PR12345 \
     --with-init-artifacts $my-local-packages/v1.12.2/
```

When reading from a local folder or from a remote repository, a `version` file should exist in the source.

Init artifacts will be placed in a well know folders, `kind/bin` and `kind/images`, and `kubelet` service
will be configured, thus making the container derived from the image ready for `kinder do kubeadm-init`
action (or for direct invocation of `kubeadm init`).

If necessary, it is possible to add more than one Kubernetes version e.g. for testing upgrade sequences.

### Add images

```bash
kinder build node-image-variant \
     --base-image kindest/node:latest \
     --image kindest/node:PR12345 \
     --with-images v1.14.0

kinder build node-image-variant \
     --base-image kindest/node:latest \
     --image kindest/node:PR12345 \
     --with-images $my-local-images/nginx.tar
```

When reading from a local folder, both single file or folder can be used as a arguments for the `--with-images`;
in case a folder is used, all the image tars existing in such folder are loaded into the node-image-variant,
thus allowing to pre-loading any image into nodes.

Image tar files will be placed in a well know folder, `kind/images` and kind(er) will load them during
the initialization of each node.

> the image tar provided to `kinder build node-image-variant` will override existing images tar with the same name;
> if necessary, the `--image-name-prefix` flag can be used to avoid name conflicts.

### Replace kubeadm/kubelet binary

```bash
kinder build node-image-variant \
     --base-image kindest/node:latest \
     --image kindest/node:PR12345 \
     --with-kubeadm v1.13.5

kinder build node-image-variant \
     --base-image kindest/node:latest \
     --image kindest/node:PR12345 \
     --with-kubeadm $working_dir/kubernetes/bazel-bin/cmd/kubeadm/linux_amd64_pure_stripped/kubeadm
```

When reading from a local folder, both single file or folder can be used; in case a folder is used, the
kubeadm binary should exist inside such folder.

Please note that, replacing the kubeadm binary in the node-images will have effect on nodes that you create in future
If you want to replace the kubeadm binary on existing nodes, you should use `docker cp` or `kinder cp` instead.

Similarly, you can use also the `--with-kubelet` flag for replacing the kubelet binary.

### Add upgrade packages

```bash
kinder build node-image-variant \
     --base-image kindest/node:latest \
     --image kindest/node:PR12345 \
     --with-upgrade-artifacts v1.14.0

kinder build node-image-variant \
     --base-image kindest/node:latest \
     --image kindest/node:PR12345 \
     --with-upgrade-artifacts $my-local-packages/v1.12.2/
```

When reading from a local folder or from a remote repository, a `version` file should exist in the source.

Upgrade artifacts for will be placed in a well know folder, `kinder/upgrade/{version}` that will be used by
`kinder do kubeadm-upgrade` action (or for direct invocation of `kubeadm upgrade`).

If necessary, it is possible to add more than one Kubernetes version e.g. for testing upgrade sequences.

### kinder get artifacts

It is also possible to get Kubernetes artifact locally using `kinder get artifacts` from one of the following sources:

- a version, e.g. v1.14.0 or v1.15.0-alpha.0.100+78573805a7292a
- a release build label, e.g. release/stable, release/stable-1.13, release/latest-14
- a ci build label, e.g. ci/latest, ci/latest-1.14
- a remote repository, e.g. <http://k8s.mycompany.com/>
- a local folder

Flags `--only-kubeadm`, `--only-kubelet`, `--only-binaries`, and `--only-images` can be used to limit the number of files read from the source.

When reading from upstream builds (version, release label, ci build label), a `version` file will be automatically
generated in the target folder.

Instead, when reading from a local folder or from a remote repository, a `version` file should exist in the source.

## Run E2E test suites

### E2E (Kubernetes)

E2E Kubernetes is a rich set of test aimed at verifying the proper functioning of a Kubernetes cluster.

By default Kinder selects a subset of test the corresponds to the "Conformance" as defined in the
Kubernetes [test grid](https://git.k8s.io/test-infra/testgrid/conformance).

```bash
kinder test e2e
```

> The test are run on the cluster defined in the current kubeconfig file

The command supports following flags:

- `--kube-root` for setting the folder where the kubernetes sources are stored
- `--conformance` as a shortcut for instructing the ginkgo test suite run only conformance tests
- `--parallel` as a shortcut for instructing the ginkgo to run test in parallel

Additional flags are supported for allowing `--ginkgo-flags` and `--test-flags` are supported for
allowing low level configuration of test runs:

- `--ginkgo-flags` space-separated list of arguments to pass to ginkgo test runner. see
  [ginkgo doc](https://onsi.github.io/ginkgo/#the-ginkgo-cli) for more detail.
- `--test-flags` space-separated list of arguments to pass to E2E test suite

Example of usage of those flags are:

```bash
# dry-run your test
kinder test e2e --ginkgo-flags "--dryDun=true"

# override the default level of parallelism
kinder test e2e --parallel --ginkgo-flags "--nodes=25"

# generate a junit report of test results
kinder test e2e --reporting-flags "--report-dir=/tmp/_artifacts --report-prefix=e2e"
```

### E2E kubeadm

Similarly to E2E Kubernetes, there is a suite of tests aimed at checking that kubeadm has created
and properly configured all the ConfigMap, Secrets, RBAC Roles and RoleBinding required for the
proper functioning of future calls to `kubeadm join` or `kubeadm upgrade`.

This can be achieved by a simple

```bash
kinder test e2e-kubeadm
```

> The test are run on the cluster defined in the current kubeconfig file

The command supports following flags:

- `--kube-root` for setting the folder where the kubernetes sources are stored
- `--single-node` as a shortcut for instructing the ginkgo test suite to skip test labeled with [multi-node]
- `--automatic-copy-certs` as a shortcut for instructing the ginkgo test suite to skip test labeled with [copy-certs]

As for E2E Kubernetes, also `--ginkgo-flags` and ``--test-flags` are supported for low
level configuration of test runs.
