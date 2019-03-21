# Kinder

kinder is an example of [kind](https://github.com/kubernetes-sigs/kind) used as a library.

All the kind commands will be available in kinder, side by side with additional commands
designed for helping kubeadm contributors.

**kinder is a work in progress. Test it! Break it! Send feedback!**

## Prerequisites

### Install git

Our source code is managed with [git](https://git-scm.com/), to develop locally you will need to install git.

You can check if git is already on your system and properly installed with the following command:

```bash
git --version
```

### Install Go

To work with kinder you will need [Go](https://golang.org/doc/install).

Install or upgrade [Go using the instructions for your operating system](https://golang.org/doc/install).
You can check if Go is in your system with the following command:

```bash
go version
```

Working with Go [modules](https://kind.sigs.k8s.io/docs/contributing/getting-started/which%20we%20use%20for%20dependency%20management) requires at least 1.11.4 due to checksum bugs
in lower versions.

## Getting started

Clone the kubeadm repository:

```bash
git clone https://github.com/kubernetes/kubeadm.git
```

And then build kinder

```bash
cd kinder
GO111MODULE=on go install
```

This will put kinder in $(go env GOPATH)/bin.

## Usage

Read the [kind documentation](https://kind.sigs.k8s.io/docs/user/quick-start/) first.

Then [Prepare for tests](doc/prepare-for-tests.md)

Follow the how to guides:

- [Getting started (test single control-plane)](doc/getting-started.md)
- [Testing HA](doc/test-HA.md)
- [Testing upgrades](doc/test-upgrades.md)
- [Testing X on Y](doc/test-XonY.md)

Or have a look at the [Kinder reference](doc/reference.md)
