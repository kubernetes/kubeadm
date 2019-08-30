#!/bin/bash

set -o nounset
set -o pipefail
set -o errexit

SEP="###############################################################################"
echo $SEP
echo -e "* running `basename "$0"`..."
echo $SEP

set -x

# install curl if missing
if ! `curl --version > /dev/null`; then
	apt-get update || exit 1
	apt-get install -y curl || exit 1
fi

# install go if missing
if ! `go version > /dev/null`; then
	apt-get update || exit 1
	apt-get install -y golang-go || exit 1
fi

LPATH=`dirname "$0"`
cd "$LPATH"

# this block acts like a cheap replacement for `godep` or `go get`.
# we only need a sub-package so no need to pull the whole apimachinery tree.
VERSION_PATH="src/k8s.io/apimachinery/pkg/util/version"
mkdir -p $VERSION_PATH
pushd $VERSION_PATH
curl -sS "https://raw.githubusercontent.com/kubernetes/apimachinery/master/pkg/util/version/version_test.go" > "version_test.go"
curl -sS "https://raw.githubusercontent.com/kubernetes/apimachinery/master/pkg/util/version/version.go" > "version.go"
popd

# set GOPATH to the local directory
export GOMODULE=on

# run unit tests
go test -v ./$VERSION_PATH/version.go ./$VERSION_PATH/version_test.go
go test -v ./verify_manifest_lists.go ./verify_manifest_lists_test.go

# run main test
go run ./verify_manifest_lists.go

# cleanup
rm -rf ./src
