#!/bin/bash

set -o nounset
set -o pipefail
set -o errexit

# This test installs debian packages that are a result of the CI and release builds.
# It runs '... --version' commands to verify that the binaries are correctly installed
# and finally uninstalls the packages.
# For the release packages it tests all versions in the support skew.

LINE_SEPARATOR="*************************************************"
echo "$LINE_SEPARATOR"

PACKAGES_TO_TEST_K8S=("kubectl" "kubelet" "kubeadm")
PACKAGES_TO_TEST_EXT=("cri-tools" "kubernetes-cni")

PACKAGES_TO_TEST=("${PACKAGES_TO_TEST_EXT[@]}" "${PACKAGES_TO_TEST_K8S[@]}")

# harcode the LSB release to "xenial" instead of calling "lsb_release -c -s"
LSB_RELEASE="xenial"

# install prerequisites

echo "* installing prerequisites"
apt-get update
apt-get install -y apt-transport-https curl python gnupg

# check if gsutil from the Google Cloud SDK exists and if not install the SDK

if ! gsutil version > /dev/null; then
  curl -sSL https://sdk.cloud.google.com > /tmp/gcl && bash /tmp/gcl --install-dir=~/gcloud --disable-prompts
  export PATH=$PATH:~/gcloud/google-cloud-sdk/bin
fi
gsutil version

# test CI packages

echo "$LINE_SEPARATOR"
echo "* TESTING CI PACKAGES"

PACKAGE_EXT="deb"
CI_DIR=/tmp/k8s-ci
mkdir -p $CI_DIR
mkdir -p /opt/cni/bin
CI_VERSION=$(curl -sSL https://dl.k8s.io/ci/latest.txt)
CI_URL="gs://kubernetes-release-dev/ci/$CI_VERSION-bazel/bin/linux/amd64"

echo "* testing CI version $CI_VERSION"

for CI_PACKAGE in "${PACKAGES_TO_TEST[@]}"; do
  echo "* downloading package: $CI_URL/$CI_PACKAGE.$PACKAGE_EXT"
  gsutil cp "$CI_URL/$CI_PACKAGE.$PACKAGE_EXT" "$CI_DIR/$CI_PACKAGE.$PACKAGE_EXT"
  dpkg -i "$CI_DIR/$CI_PACKAGE.$PACKAGE_EXT" || echo "* ignoring expected 'dpkg -i' result"
done

echo "* installing"

apt-get install -f -y

echo "* testing binaries"

kubeadm version -o=short
kubectl version --client=true --short=true
kubelet --version
crictl --version
ls /opt/cni/bin/loopback

echo "* removing packages"

apt-get remove -y --purge "${PACKAGES_TO_TEST[@]}"
apt-get autoremove -y

rm -rf $CI_DIR

# test stable packages
# NOTE: MAJOR version bumps are not supported!

echo "$LINE_SEPARATOR"
echo "* TESTING STABLE PACKAGES"

curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
cat <<EOF > /etc/apt/sources.list.d/kubernetes.list
  deb http://apt.kubernetes.io/ kubernetes-$LSB_RELEASE main
EOF
apt-get update

VERSIONS=$(apt-cache madison kubelet | awk '{print $3}')
LATEST_STABLE=$(apt-cache madison kubelet | awk '{print $3}' | head -1)
LATEST_STABLE_MINOR=$(echo "$LATEST_STABLE" | cut -d '.' -f 2)
MIN_SUPPORTED_MINOR=$(("$LATEST_STABLE_MINOR" - 2))

for VER in $VERSIONS; do
  echo "$LINE_SEPARATOR"
  echo "* testing version: $VER"
  MINOR=$(echo "$VER" | cut -d '.' -f 2)

  if [[ "$MINOR" -lt "$MIN_SUPPORTED_MINOR" ]]; then
    echo "* found version outside of the support skew: $VER; finishing..."
    break
  fi

  # add versions for the k/k originated packages
  PACKAGES_TO_TEST_K8S_VER=()
  for PKG_K8S in "${PACKAGES_TO_TEST_K8S[@]}"; do
    PACKAGES_TO_TEST_K8S_VER+=("$PKG_K8S=$VER")
  done

  echo "* installing ${PACKAGES_TO_TEST_EXT[*]} ${PACKAGES_TO_TEST_K8S_VER[*]}"

  apt-get install -y "${PACKAGES_TO_TEST_EXT[@]}" "${PACKAGES_TO_TEST_K8S_VER[@]}"

  echo "* testing binaries"

  kubeadm version -o=short
  kubectl version --client=true --short=true
  kubelet --version
  crictl --version
  ls /opt/cni/bin/loopback

  echo "* removing packages"

  apt-get remove -y --purge "${PACKAGES_TO_TEST[@]}"
  apt-get autoremove -y
done
