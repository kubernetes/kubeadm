# Kubeadm Playground

The kubeadm playground consists of one or more vagrant machines where you can experiment
kubeadm and eventually test a local build of kubeadm itself.

<!-- TOC -->

- [Kubeadm Playground](#kubeadm-playground)
    - [Pre-requisites](#pre-requisites)
    - [Installation](#installation)
    - [Getting started](#getting-started)
        - [Create the kubeadm test playground](#create-the-kubeadm-test-playground)
        - [Deploy local build of kubeadm to the playground](#deploy-local-build-of-kubeadm-to-the-playground)
        - [Access the playground](#access-the-playground)
    - [Iterative dev/build/release/test cycles](#iterative-devbuildreleasetest-cycles)
    - [Advanced options](#advanced-options)
        - [Customize the kubeadm playground](#customize-the-kubeadm-playground)
        - [Customizing the playground provisioning process](#customizing-the-playground-provisioning-process)
        - [Executing kubeadm E2E tests manually](#executing-kubeadm-e2e-tests-manually)
        - [Executing kubeadm E2E tests automatically](#executing-kubeadm-e2e-tests-automatically)
        - [Testing pre-release versions](#testing-pre-release-versions)
            - [inject pre-release versions during kubeadm-playground start](#inject-pre-release-versions-during-kubeadm-playground-start)
            - [inject pre-release versions with kubeadm-playground deploy](#inject-pre-release-versions-with-kubeadm-playground-deploy)
        - [Comparing two version of kubeadm](#comparing-two-version-of-kubeadm)
        - [create a base box](#create-a-base-box)

<!-- /TOC -->

## Pre-requisites

- vagrant
- ansible (_if ansible is not installed, Kubeadm Playground will work with limited functionalities_)
- A working kubernetes dev environment (_only if you are testing local build of kubeadm_).

## Installation

```bash
cd ~/go/src/k8s.io
git clone https://github.com/kubernetes/kubeadm.git
```

In order to make the `kubeadm-playground` command easily accessible from other locations,
open your `.bashrc` file and append the following line:

```bash
alias kubeadm-playground='~/go/src/k8s.io/kubeadm/vagrant/kubeadm-playground'
```

## Getting started

### Create the kubeadm test playground

Execute:

```bash
kubeadm-playground start
```

And, by default, a vagrant machine will be provisioned, with all the requisites
for executing `kubeadm init` in place.

### Deploy local build of kubeadm to the playground

In case you want to test local release of kubeadm :

- Build kubeadm from source:

```bash
cd ~/go/src/k8s.io/kubernetes

# run all unit tests
bazel test //cmd/kubeadm/...

# build kubeadm binary for the target platform used by the the machines in the playground (linux)
bazel build //cmd/kubeadm --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64
```

- Deploy the kubeadm binary to the playground with:

```bash
kubeadm-playground deploy
```

As a result the kubeadm version will be accessible by the playground machines into the `/vagrant/bin` folder.

### Access the playground

Access the playground with:

```bash
kubeadm-playground ssh kubeadm-test-master
```

And then execute your kubeadm test scenario (See following paragraphs for more info).

## Iterative dev/build/release/test cycles

If you are developing new kubeadm features, most probably you will execute several dev/build/release/test
cycles.

In case it is not necessary to fully recreate the kubeadm playground before a new test session,
run `kubeadm reset` on all machines (either manually by ssh-ing into each machines or
automatically invoking `kubeadm-playground exec reset` from the host machine)

Instead, in case it is necessary to recreate the kubeadm playground before a
new test session, run `kubeadm-playground delete` and then `kubeadm-playground start` again.

Afterwards, repeat build, deploy and test steps as described above.

## Advanced options

### Customize the kubeadm playground

By default the kubeadm playground is composed by a single machine with role master.

It is possible to customize the kubeadm playground by modifying the cluster definition in
the `spec/` folder or creating your own spec folder and passing it to `kubeadm-playground`
with the `--spec` flag or the `KUBEADM_PLAYGROUND_SPEC` environment variable.

Supported options are:

- Different cluster topologies e.g. add master nodes, add worker nodes, define the
  nodes in charge for hosting an external etcd cluster.

- Different attributes of the cluster, selecting among all the supported kubeadm options. You can define:
  - The target kubernetes/kubeadm version (including CI/CD versions)
  - How the controlplane will be deployed (static pods vs self hosting)
  - Which type of certificate authority are you going to use (local vs external)
  - Where your PKI will be stored (filesystem or secrets)
  - Which type of DNS are you going to use (kubeDNS or coreDNS)
  - Which type of pod network add-on are you going to use (weavenet, flannel, calico)

See the [readme](spec/README.md) file under `spec/` folder for more info

### Customizing the playground provisioning process

`kubeadm-playground start` by default will install all the necessary pre-requisites for
running `kubeadm init`.

If ansible is not installed on the guest machine, this includes only kubernetes, kubelet and
kubeadm binaries, that is basically what you need for test in clusters with only one machine.

Instead, if ansible is available `kubeadm-playground start` by default will take charge of
executing following additional steps simplifying test activities and making possible also
scenarios with more than on machine:

- Additional kubernetes, kubelet configurations required for running in vagrant (e.g. node-ip)
- Creation of a kubeadm config file customized with the selected kubeadm options and saved
  in `/etc/kubernetes/kubeadm.conf`
- Creation of an external certificate authority if required by cluster options
- Installation of an external vip/load balancer if the cluster has more than one machine
  with role `Master`
- Installation of an external etcd if the cluster has at least one machine with role `Etcd`

The default ansible behavior can be changed by providing by passing a list of ansible
playbooks to the `kubeadm-playground start` command; available playbooks are:

- `prerequisites` (kubernetes, kubelet, kubeadm binaries + config)
- `external-etcd`
- `external-ca`
- `external-vip`
- `kubeadm-config`
- `kubeadm-init`
- `kubectl-apply-network`
- `kubeadm-join`
- `all` (default playbooks + `kubeadm-init`, `kubectl-apply-network` and `kubeadm-join`
  if there are machine with role node)
- `none` (none the above)

Accordingly, it is possible reduce actions executed by `kubeadm-playground start`, e.g.

```bash
# install only prerequisites (but not external etcd, external ca etc.)
kubeadm-playground start prerequisites
```

But it is also possible to extend actions executed automatically by `kubeadm-playground start` up
to obtaining a fully working kubernetes cluster, e.g.

```bash
# install everything required on a single node cluster, and execute kubeadm init and join
kubeadm-playground start prerequisites kubeadm-config kubeadm-init kubectl-apply-network kubeadm-join
```

### Executing kubeadm E2E tests manually

`kubeadm-playground start` by default will install all the necessary pre-requisites
for running `kubeadm init`.

Running `kubeadm init` and eventually `kubeadm join` on nodes is typically up to the user
as initial part of your test scenario (use `kubeadm-playground ssh` to connect to the machines).

A getting started guide for executing common kubeadm actions is accessible by running `kubeadm-playground help`
followed by one of the actions described in the previous paragraph e.g.

```bash
kubeadm-playground help kubeadm-init
```

If ansible is available on the guest machine, it is also possible to execute single provisioning steps automatically e.g.

```bash
kubeadm-playground exec kubeadm-init
```

### Executing kubeadm E2E tests automatically

kubeadm playground include the necessary scaffholding code for automating kubeadm E2E tests (e.g upgrades sequences)
by developing an ansible playbooks. Test will be afterwards available via the `e2e` command e.g.

```bash
kubeadm-playground e2e test-example
```

### Testing pre-release versions

Kubeadm-playground by default uses the `kubernetes.version` field in the cluster API definition as a source for
the version of kubeadm, kubelet, kubectl .deb or .rpm and for the version of the controlplane components to be installed.

While `kubernetes.version`  can be used only with GA versions, kubeadm-playground allows to test alpha, beta, CI/CD
or local build versions. See [Testing pre-release versions of Kubernetes with kubeadm](https://github.com/kubernetes/kubeadm/blob/master/testing-pre-releases.md)
for more details about how to retrieve/how to build pre-release versions.

kubeadm-playground allows to:

- "inject" pre-release version during `kubeadm-playground start` (.deb packages, bynaries, tar of docker images)
- "inject" pre-release versions of kubeadm binary on an already existing playground with `kubeadm-playground deploy`
  (used for iterative dev/build/release/test cycles)

#### inject pre-release versions during kubeadm-playground start

`kubeadm-playground start` provide support automatic deployment of pre-release artifacts; to trigger this feature
you should specify a folder where pre-release packages are available by using the `--packages` flag or the
`KUBEADM_PLAYGROUND_PACKAGES` environment variable.

Deployment of pre-release artifacts depends by the content of the packages folder.

- If the packages folder contains `kubectl.deb`, `kubernetes-cni.deb`, `kubelet.deb` or `kubeadm.deb`,
  such packages will installed instead of the corresponding package from the official kubernetes deb repository
- If the packages folder contains `kubectl` or `kubelet` binary, such binary will override the corresponding binary
  installed by .deb packages into the `/usr/bin/` folder.
- If the packages folder contains tars with docker images, those tars will be loaded into docker.

#### inject pre-release versions with kubeadm-playground deploy

`kubeadm-playground deploy` command will copy the kubeadm binary contained in the bazel output folder on the
local machine into the `/vagrant/bin` folder of all the playground machines.

> NB. kubeadm binary injected with deploy into the `/vagrant/bin` folder does not override the kubeadm binary
installed during `kubeadm-playground start` into the `/usr/bin/` folder; to access the injected kubeadm binary
you must specify the full path e.g. `/vagrant/bin/kubeadm`.

Following options are available to customize the deploy action:

- use the `--binary` flag or the `KUBEADM_PLAYGROUND_BINARY` environment variable to
  specify a local kubeadm binary to be deployed to the vagrant playground
- use the `--builder` flag or the `KUBEADM_PLAYGROUND_BUILDER` environment variable to
  specify the kubernetes build method in use (choices are `bazel`, `docker` or `local`, default `bazel`);
  the kubeadm binary available in the build output will be deployed to the vagrant playground

Additionally, it is possible to specify a prefix for the target kubeadm binary file name using the `--issue`
or `--pr` flags or the `KUBEADM_PLAYGROUND_ISSUE` environment variable.

### Comparing two version of kubeadm

In some case it is useful to compare the output of two versions of kubeadm.

To setup the kubeadm  playground for this use cases you can invoke `kubeadm-playground deploy`
two times by passing two different `--issue` or `--pr` numbers. As a result both the kubeadm version
will be accessible by the vagrant machines into the `/vagrant/bin` folder, each one with the given
issue/pr number as a prefix.

Then you can connect to the playground with `kubeadm-playground ssh` and:

```bash
  # use the first version of kubeadm e.g. deployed with prefix 594
  sudo /vagrant/bin/594_kubeadm init

  # reset to a clean state
  sudo /vagrant/bin/594_kubeadm reset

  # compare with the second version of kubeadm e.g. deployed with prefix 710
  sudo /vagrant/bin/710_kubeadm init
```

### create a base box

In order to make working `kubeadm-playground` provisioning faster, you can create a
vagrant base box with all the prerequisites already in place.
see [instruction](box.md) for more info
