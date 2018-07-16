# Contributing guidelines

## How to become a contributor and submit your own code

### Contributor License Agreements

We'd love to accept your patches! Before we can take them, we have to jump a couple of legal hurdles.

Please fill out either the individual or corporate Contributor License Agreement (CLA).

  * If you are an individual writing original source code and you're sure you own the intellectual property, then you'll need to sign an [individual CLA](https://identity.linuxfoundation.org/node/285/node/285/individual-signup).
  * If you work for a company that wants to allow you to contribute your work, then you'll need to sign a [corporate CLA](https://identity.linuxfoundation.org/node/285/organization-signup).

Follow either of the two links above to access the appropriate CLA and instructions for how to sign and return it. Once we receive it, we'll be able to accept your pull requests.

### Contributing A Patch

1. Submit an issue describing your proposed change to the repo in question.
2. The [repo owners](OWNERS) will respond to your issue promptly.
3. If your proposed change is accepted, and you haven't already done so, sign a Contributor License Agreement (see details above).
4. Fork the desired repo, develop and test your code changes.
5. Submit a pull request.

### Contributing kubeadm documentation

`kubeadm` is documented in various places on the [kubernetes.io](https://kubernetes.io/docs/search/?q=kubeadm) website.
These pages cover topics such as installation steps, troubleshooting and command line syntax.
You can help `kubeadm` **a lot** by filling issue reports for inconsistencies and keeping the documentation up-to-date by submitting PRs.

The process for contributing to the website is very straight forward and is outlined here:
* https://github.com/kubernetes/website/blob/master/CONTRIBUTING.md

Here is a document that explains the process of updating the `kubeadm` command line reference:
* https://github.com/kubernetes/kubeadm/blob/master/docs/updating-command-reference.md

### Building

`kubeadm` uses the same build process as the rest of the `kubernetes/kubernetes` repository.
However, you do not frequently have to build all of kubernetes to work on kubeadm.

See [./vagrant/README.md](./vagrant/README.md) for a quick workflow to build and test your own kubeadm binaries. 🙂

### Testing pre-release versions of Kubernetes with kubeadm

See [testing-pre-releases.md](testing-pre-releases.md) for information about how to get pre-release versions of kubernetes
or kubeadm and how to test them.

### Adding dependencies

If your patch depends on new packages, add that package with [`godep`](https://github.com/tools/godep). Follow the [instructions to add a dependency](https://github.com/kubernetes/kubernetes/blob/master/docs/devel/development.md#godep-and-dependency-management).

### Running unit tests

First navigate to the folder where you have cloned kubernetes (e.g. `~/go/src/k8s.io/kubernetes`).

To run `kubeadm` unit tests for the `cmd/kubeadm/app/cmd` package call:
```
./hack/make-rules/test-kubeadm-cmd.sh
```

You can also run unit tests for specific `kubeadm` packages using:
```
make test WHAT=<package> GOFLAGS="-v"
```
Where `<package>` can be `./cmd/kubeadm/app/cmd`, `./cmd/kubeadm/app/utils`, `./cmd/kubeadm/app/features`, etc.

For more information about running tests in Kubernetes have a look at:
* https://github.com/kubernetes/community/blob/master/contributors/devel/testing.md

For more general information about unit tests in Go please have a look at:
* https://golang.org/pkg/testing/
* https://blog.alexellis.io/golang-writing-unit-tests/
