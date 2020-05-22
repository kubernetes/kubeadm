# Prepare for tests

Before starting test with kinder, it is necessary to get a node-image to be used as a base for nodes in the cluster.

As in kind, also in kinder in order to make your test will fast and repeatable, it is recommended to
pack whatever you need during your test in the node-images.

kind gives already you what you need in most cases (kubeadm, Kubernetes binaries, pre-pulled images); kinder
on top of that allows to build node variants for addressing following use cases:

- Adding a Kubernetes version to be used for initializing the cluster (as an alternative to `build node-image` already supported by kind)
- Adding new pre-loaded images that will be made available on all nodes at cluster creation time
- Replacing the kubeadm binary installed in the cluster, e.g. with a locally build version of kubeadm
- Replacing the kubelet binary installed in the cluster, e.g. with a locally build version of kubelet
- Adding a second Kubernetes version to be used for upgrade testing

## Use a kind public node-images

The easiest way to get a node image for a major/minor Kubernetes version is use kind images available
on docker hub, e.g.

```bash
docker pull kindest/node:vX
```

## Build a node-image

Build a base-image (or download one from docker hub)

```bash
kinder build base-image --image kindest/base:latest
```

Build a node-image starting from the above base image:

```bash
kind build node-image --base-image kindest/base:latest --image kindest/node:vX
```
> NB see [kind documentation](https://kind.sigs.k8s.io/docs/user/quick-start/#building-images) for more information about `kind build node-image` usage.

As an alternative, it is possible to pick an existing base image and customize it by adding a Kubernetes
version with:

```bash
kinder build node-image-variant \
     --base-image kindest/base:vX \
     --image kindest/node:vX-variant \
     --with-init-artifacts $mylocalbinary/kubeadm
```

Please note that `kinder build node-image-variant` accepts as input:

- a version, e.g. v1.14.0
- a release build label, e.g. release/stable, release/stable-1.13, release/latest-14.
- a ci build label, e.g. ci/latest, ci/latest-14.
- a remote repository, e.g. <http://k8s.mycompany.com/>
- a local folder, as shown in the examples above.

It is also possible to get Kubernetes artifacts locally using `kinder get artifacts`.

See [Kinder reference](reference.md) for more detail.

## Customize a node-image

As a third option for building node-image, it is possible to pick an existing node image and customize it by:

1. overriding the kubeadm binary

```bash
kinder build node-image-variant \
     --base-image kindest/node:vX \
     --image kindest/node:vX-variant \
     --with-kubeadm $mylocalbinary/kubeadm
```

1. adding/overriding the pre loaded images in the `/kind/images` folder

```bash
kinder build node-image-variant \
     --base-image kindest/node:vX \
     --image kindest/node:vX-variant \
     --with-images $mylocalimages/nginx.tar
```

1. adding a second Kubernetes version in the `/kinder/upgrades` folder for testing upgrades

```bash
kinder build node-image-variant \
     --base-image kindest/node:vX \
     --image kindest/node:vX-variant \
     --with-upgrade-artifacts $mylocalbinaries/vY
```

Please note that `kinder build node-image-variant` accepts as input:

- a version, e.g. v1.14.0
- a release build label, e.g. release/stable, release/stable-1.13, release/latest-14.
- a ci build label, e.g. ci/latest, ci/latest-14.
- a remote repository, e.g. <http://k8s.mycompany.com/>
- a local folder, as shown in the examples above.

It is also possible to get Kubernetes artifacts locally using `kinder get artifacts`.

See [Kinder reference](reference.md) for more detail.
