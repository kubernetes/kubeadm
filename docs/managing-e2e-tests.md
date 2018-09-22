
## Managing kubeadm end-to-end tests

### Overview

Kubernetes has a rich end-to-end (e2e) testing infrastructure, which allows detailed testing of clusters and their assets. Most settings and tools for that can be found in the [test-infra](https://github.com/kubernetes/test-infra) GitHub repository.

Kubernetes uses applications such as the web-based [testgrid](https://k8s-testgrid.appspot.com/) for monitoring the status of e2e tests. test-infra also hosts the configuration on individual test jobs and Docker images that contain tools to invoke the jobs.

### Details on kubeadm test jobs

The following folder contains all the SIG Cluster Lifecycle (the SIG that maintains kubeadm) originated test jobs:
[sig-cluster-lifecycle](https://github.com/kubernetes/test-infra/tree/master/config/jobs/kubernetes/sig-cluster-lifecycle)

Please note that this document will only cover details on the `kubeadm*.yaml` files.

For example, let's have a look at this file:
[kubeadm.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm.yaml)

It contains a list of jobs such as:
```
- name: ci-kubernetes-e2e-kubeadm-gce-master
  interval: 2h
  ...
```

In this case, `ci-kubernetes-e2e-kubeadm-gce-master` is a test job that runs every 2 hours and it also has a set of parameters defined for it. This document will only cover some of them.

Important to note here is that testgrid jobs have a docker `image` that contains all the tools needed to run tests. Among them, one of the most relevant ones, is the tool that will be used for the deployment of the cluster.

For this example job the `deployment` tool is called [kubernetes-anywhere](https://github.com/kubernetes/kubernetes-anywhere) (with integration in [this file](https://github.com/kubernetes/test-infra/blob/master/kubetest/anywhere.go)) and the `image` is called `gcr.io/k8s-testimages/e2e-kubeadm:...`. As a very high level summary, the way this works is when a job is invoked all the job parameters are passed to a CLI tool called `kubetest`, which then instantiates the job container and then the deployment tool inside it.

Please note that there are plans for replacing kubernetes-anywhere with a new tool based on the [Cluster API](https://github.com/kubernetes-sigs/cluster-api).

### The testgrid configuration

testgrid uses a rather large configuration file that hosts the dashboard of all the tests that can be found [here](https://k8s-testgrid.appspot.com). The initial list of menu items mostly holds Kubernetes SIGs, while the SIG Cluster Lifecycle specific dashboard can be found [here](https://k8s-testgrid.appspot.com/sig-cluster-lifecycle-all).

The configuration file itself is located [here](https://github.com/kubernetes/test-infra/blob/master/testgrid/config.yaml). The file is broken into multiple sections. If you wish to find the SIG Cluster Lifecycle and kubeadm tests you can search for `sig-cluster-lifecycle`. For instance, `sig-cluster-lifecycle-all` has all the relevant kubeadm tests.

A dashboard is defined like so:
```
- name: sig-cluster-lifecycle-all
  dashboard_tab:
  - name: kubeadm-gce-1.10
    test_group_name: ci-kubernetes-e2e-kubeadm-gce-1-10
  - name: kubeadm-gce-1.11
    test_group_name: ci-kubernetes-e2e-kubeadm-gce-1-11
  ...
```

What the above does is defining a dashboard called `sig-cluster-lifecycle-all` and placing inside a couple of tests defined by `name` and linking to a `test_group_name`. Some items may also have a `description`.

Note that `test_group_name` represents the actual test name (as in the [kubeadm.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm.yaml) example earlier), while `name` is only what is shown on the website as an alias.

Tests can exist in multiple dashboards at once. For example a test defined like:
```
- name: kubeadm-gce-1.12
  test_group_name: ci-kubernetes-e2e-kubeadm-gce-1-12
```
can exists in both the `sig-cluster-lifecycle-all` and `sig-release-master-blocking` dashboards. Essentially both tests would show details about the same test job called `ci-kubernetes-e2e-kubeadm-gce-1-12`.

### Updates to kubeadm tests

Before each new Kubernetes release and during the second month of the [release-cycle](https://github.com/kubernetes/kubeadm/blob/master/docs/release-cycle.md), a set of manual actions have to be performed, so that the kubeadm e2e tests are up to date with the new release.

The operation can be broken down in 3 steps:
1) Updating the kubeadm test jobs and parameters.
2) Updating testgrid to reflect the changes.
3) Finalizing the changes.

#### Updating test jobs

This document will cover information on how to perform updates on these 3 files:
- [kubeadm.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm.yaml "kubeadm.yaml")

Holds test jobs where the kubeadm version matches the Kubernetes control-plane and the kubelet versions.

- [kubeadm-upgrade.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm-upgrade.yaml "kubeadm-upgrade.yaml")

Holds test jobs that perform a upgrade from version X to version Y (usually `Y = X + 1`).

- [kubeadm-x-on-y.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm-x-on-y.yaml "kubeadm-x-on-y.yaml")

Holds test jobs that run kubeadm version Y, on a control plane and kubelet version X,
as kubeadm does support `Y = X + 1`

The files can be found in the [sig-cluster-lifecycle](https://github.com/kubernetes/test-infra/tree/master/config/jobs/kubernetes/sig-cluster-lifecycle) folder.

The idea is to remove old test jobs and add new ones. If Kubernetes 1.15 is about to be released soon, this means that the `15 - 3 = 12` MINOR version would go outside of the support skew of the project. `1.12` jobs have to be removed and `1.15` jobs have to be added. The project would support version `15 - 2 = 13` as the minimum.

- If there are `1.12 -> 1.13` upgrade jobs or `1.12 on 1.13` jobs leave them, because they still contain a supported version `1.13`.
- Remove plain `1.12` jobs.
- Add upgrade jobs for `1.14 -> 1.15` and `1.14 on 1.15` jobs.
- Add plain `1.15` jobs.

Each job contains some versioned labels and branches passed as properties, that are not going to be outlined in this document, but they need updates too:
- `release-1.14`
- `release/latest-1.14`
- `ci/latest-bazel-1.14`
- `latest-bazel-1.14`
- ...

Example PRs can be found [here](https://github.com/kubernetes/test-infra/pull/9219) and [here](https://github.com/kubernetes/test-infra/pull/8142).

#### Updating testgrid to reflect the changes

Once the changes to the jobs have been made it is time to open the testgrid [configuration file](https://github.com/kubernetes/test-infra/blob/master/testgrid/config.yaml) and modify it as well .

The same procedure applies here - old jobs are being removed and new jobs are being added.

Search and replace is your friend. Remember the job names that you just changed in the the `kubeadm*.yaml` files. Don't forget that jobs can exist under multiple dashboards in testgrid, so make sure you find all occurrences and modify them.

Following the `1.15` example from earlier, a job called `ci-kubernetes-e2e-kubeadm-gce-1-12` would have to be removed from all occurrences and a new job `ci-kubernetes-e2e-kubeadm-gce-1-15` has to be added in it's place.

Removed:
```
- name: kubeadm-gce-1.12
  test_group_name: ci-kubernetes-e2e-kubeadm-gce-1-12
```

Added:
```
- name: kubeadm-gce-1.15
  test_group_name: ci-kubernetes-e2e-kubeadm-gce-1-15
```

Example PRs can be found [here](https://github.com/kubernetes/test-infra/pull/9219) and [here](https://github.com/kubernetes/test-infra/pull/8142).

#### Adding release blocking jobs

Late in the Kubernetes release cycle a new branch is created for the new release in the `kubernetes/kubernetes` repository. Following our `1.15` example the branch would be called `release-1.15`. In testgrid, a couple of new dashboards are created and maintained by SIG Release. These dashboards are:
- `sig-release-1.15-all`
- `sig-release-1.15-blocking`

The kubeadm project has to add some jobs to these dashboards which are important for the Kubernetes release. The `sig-release-1.15-blocking` dashboard in particular would contain jobs, that might delay a new `1.15` release if they are not passing (red).

The list of `blocking` jobs for the `1.15` example would be:
- `ci-kubernetes-e2e-kubeadm-gce-1-14-on-1-15`
- `ci-kubernetes-e2e-kubeadm-gce-1-15`

The list of `all` jobs for the `1.15` example would be:
- `ci-kubernetes-e2e-kubeadm-gce-1-14-on-1-15`
- `ci-kubernetes-e2e-kubeadm-gce-1-15`
- `ci-kubernetes-e2e-kubeadm-gce-upgrade-1-14-1-15`

[Here](https://github.com/kubernetes/test-infra/pull/9529/files) is an example DIFF of a GitHub PR that makes the outlined changes.

Note that these release dashboards remain until version `1.15` goes outside of the support skew, or that would be when `1.18` is released.

#### Finalizing the changes.

Once you have made the above test-infra changes, you need to run a certain script to reflect the changes in generated configuration. For that you can call the following command:

```
./hack/update-all.sh
```

You can also run verification for your changes by calling the verification scripts:
```
./hack/verify-*.sh
```

Or building test-infra:
```
make
```
Note that test-infra requires [Bazel](https://bazel.build/).

At this point you can commit your changes and send a GitHub PR against the test-infra repository.

#### Resources

- Here is a pair-programming [video session](https://www.youtube.com/watch?v=aTGbdPU0fE8) between Lucas Käldström and Hippie Hacker covering the same process for the Kubernetes 1.11 release.
