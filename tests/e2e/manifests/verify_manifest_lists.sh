#!/bin/bash

set -o nounset
set -o pipefail
set -o errexit

SEP="###############################################################################"
echo $SEP
echo -e "* running `basename "$0"`..."
echo $SEP

set -x

# cleanup go.* on exit
trap "rm go.mod go.sum" EXIT

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

# use go modules. this forces using the latest k8s.io/apimachinery package.
export GO111MODULE=on
go mod init verify-manifest-lists

# run unit tests
go test -v ./verify_manifest_lists.go ./verify_manifest_lists_test.go

# run main test
go run ./verify_manifest_lists.go

# cleanup
rm -rf ./src
