#!/bin/sh

# copy_kubeadm_bin takes your built kubeadm binary, prefixes it with an issue
# number and copies it into this repo's /bin, so that it may be used from
# `/vagrant/bin` inside the VM for testing.
#
#   usage:
#     issue=710 ./copy_kubeadm_bin.sh
#   env_vars:
#     issue (*required)     Issue number(s) or descriptor used for prefixing the binary
#     kube_root | GOPATH    Used for setting the kubernetes source root. (defaults to ~/go)
#     builder               Used for selecting the host binpath. Can be one of [bazel|docker] (defaults to bazel)

set -eu
binary=kubeadm
kube_root="${kube_root:-${GOPATH:-${HOME}/go}/src/k8s.io/kubernetes}"
vagrant_root="$(cd "$(dirname "${BASH_SOURCE[0]:-$PWD}")" 2>/dev/null 1>&2 && pwd)"
builder="${builder:-bazel}"

bazel_binpath="bazel-bin/cmd/kubeadm/linux_amd64_pure_stripped/"
docker_binpath="_output/local/bin/linux/amd64/"

case "${builder}" in
  docker) binpath="${docker_binpath}" ;;
  bazel) binpath="${bazel_binpath}" ;;
esac

mkdir -p ${vagrant_root}/bin

set -x
cd "${kube_root}/${binpath}"
chmod 755 ${binary}
cp ${binary} ${issue}_${binary}
cp ${binary} ${vagrant_root}/bin/${issue}_${binary}
