# Kinder

kinder is an example of [kind](https://github.com/kubernetes-sigs/kind) used as a library.

All the kind commands will be available in kinder, side by side with additional commands
designed for helping kubeadm contributors.

**kinder is a work in progress. Test it! Break it! Send feeback!**

## Prerequisites

### Install git

Our source code is managed with [git](https://git-scm.com/), to develop locally you will need to install git.

You can check if git is already on your system and properly installed with the following command:

```bash
git --version
```

### Install Go

To work on kind’s codebase you will need [Go](https://golang.org/doc/install).

Install or upgrade [Go using the instructions for your operating system](https://golang.org/doc/install).
You can check if Go is in your system with the following command:

```bash
go version
```

Working with Go [modules](https://kind.sigs.k8s.io/docs/contributing/getting-started/which%20we%20use%20for%20dependency%20management) requires at least 1.11.4 due to checksum bugs
in lower versions.

## Getting started

Read the [kind documentation](https://kind.sigs.k8s.io/docs/user/quick-start/) first, becuase kinder
is built using kind as a library.

Clone the kubeadm repo:

```bash
git clone https://github.com/kubernetes/kubeadm.git
```

And then build kinder

```bash
cd kinder
GO111MODULE=on go install
```

This will put kinder in $(go env GOPATH)/bin.

## Create a test cluster

You can create a cluster in kinder using `kinder create cluster`, that is a wrapper on the `kind create cluster` command.

However, additional flags are implemented for enabling following use cases:

### Create *only* nodes

By default kinder stops the cluster creation process before executing `kubeadm init` and `kubeadm join`;
This will give you nodes ready for intalling Kubernetes and more specifically

- the necessary prerequisited already installed on all nodes
- a pre-build kubeadm config file in `/kind/kubeadm.conf`
- in case of more than one control-plane node exists in the cluster, a pre-configured external load balancer

If instead you want to revert to the default kind behaviour, you can use the `--setup-kubernetes`:

```bash
kinder create cluster --setup-kubernetes=true
```

### Testing different cluster topologies

You can use the `--control-plane-nodes <num>` flag and/or the `--worker-nodes <num>`  flag
as a shurtcut for creating different cluster topologies. e.g.

```bash
# create a cluster with two worker nodes
kinder create cluster --worker-nodes=2
  
# create a cluster with two control-pane nodes
kinder create cluster ---control-plane-nodes=2
```

Please note that a load balancer node will be automatically create when there are more than
one control-plane node; if necessary, you can use `--external-load balancer` flag to explicitly
request the creation of an external load balancer node.

More sophisticated cluster topologies can be achieved using the kind config file, like e.g. customizing
kubeadm-config or specifying volume mounts. see [kind documentation](https://kind.sigs.k8s.io/docs/user/quick-start/#configuring-your-kind-cluster)
for more details.

### Testing Kubernetes cluster variants

kinder gives you shortcuts for testing Kubernetes cluster variants supported by kubeadm:

```bash
# create a cluster using kube-dns instead of CoreDNS
kinder create cluster --kube-dns

# create a cluster using an external etcd
kinder create cluster --external-etcd
```

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
| kubeadm-init    | Executes the kubeadm-init workflow, installs the CNI plugin and then copies the kubeconfig file on the host machine. Available options are:<br /> `--use-phases` triggers execution of the init workflow by invoking single phases. <br />`--automatic-copy-certs` instruct kubeadm to use the automatic copy cert feature.|
| manual-copy-certs      | Implement the manual copy of certificates to be shared acress control-plane nodes (n.b. manual means not managed by kubeadm) Available options are:<br />  `--only-node` to execute this action only on a specific node. |
| kubeadm-join    | Executes the kubeadm-join workflow both on secondary control plane nodes and on worker nodes. Available options are:<br /> `--use-phases` triggers execution of the init workflow by invoking single phases.<br />`--automatic-copy-certs` instruct kubeadm to use the automatic copy cert feature.<br /> `--only-node` to execute this action only on a specific node. |
| kubeadm-upgrade |Executes the kubeadm upgrade workflow and upgrading K8s. Available options are:<br /> `--upgrade-version` for defining the target K8s version.<br />`--only-node` to execute this action only on a specific node.                                             |
| Kubeadm-reset   | Executes the kubeadm-reset workflow on all the nodes. Available options are:<br />  `--only-node` to execute this action only on a specific node. |
| cluster-info    | Returns a summary of cluster info including<br />- List of nodes<br />- list of pods<br />- list of images used by pods<br />- list of etcd members |
| smoke-test      | Implements a non-exhaustive set of tests that aim at ensuring that the most important functions of a Kubernetes cluster work |

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
| @cpN     | the secondary master nodes                                   |
| @w*      | all the worker nodes                                         |
| @lb      | the external load balancer                                   |
| @etcd    | the external etcd                                            |

As alternative to node selector, the node name (the container name without the cluster name prefix) can be used to target actions to a specific node.

```bash
# run kubeadm join on the first worker node node only
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

> Please note that,  `docker cp` or `kinder cp`  allows you to replace the kubeadm kinary on existing nodes. If you want to replace the kubeadm binary on nodes that you create in future, please check altering node images paragraph

## Altering node images

Kind can be estremely efficient when the node image contains all the necessary artifacts.

kinder allows kubeadm contributor to exploit this feature by implementing the `kinder build node-variant` command, that takes a node-image and allows to build variants by:

- Adding new pre-loaded images that will be made available on all nodes at cluster creation time
- Replacing the kubeadm binary installed in the cluster, e.g. with a locally build version of kubeadm
- Adding deb packages for a second Kubernetes version to be used for upgrade testing

 The above options can be combined toghether in one command, if necessary

### Add images

```bash
kinder build node-variant \
     --base-image kindest/node:latest \
     --image kindest/node:PR12345 \
     --with-images $my-local-images/nginx.tar
```

Both single file or folder can be used as a arguments for the `--with-images`, but only image tar files will be considered; Image tar file will be placed in a well know folder, and kind(er) will load them during the initialization of each node.

### Replace kubeadm binary

```bash
kinder build node-variant \
     --base-image kindest/node:latest \
     --image kindest/node:PR12345 \
     --with-kubeadm $working_dir/kubernetes/bazel-bin/cmd/kubeadm/linux_amd64_pure_stripped/kubeadm
```

> Please note that, replacing the kubeadm binary in the node-images will have effect on nodes that you create in future; If you want to replace the kubeadm kinary on existing nodes, you should use `docker cp` or `kinder cp` instead.

### Add upgrade packages

```bash
kinder build node-variant \
     --base-image kindest/node:latest \
     --image kindest/node:PR12345 \
     --with-upgrade-packages $my-local-packages/v1.12.2/
```

Both single file or folder can be used as a arguments for the `--with-upgrade-packages`, but only deb packages will be considered; deb files will be placed in a well know folder, the kubeadm-upgrade action will use them during the upgrade sequence.