#!/bin/bash
apt-get update && apt-get install -y apt-transport-https curl
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
cat <<EOF >/etc/apt/sources.list.d/kubernetes.list
deb http://apt.kubernetes.io/ kubernetes-xenial main
EOF
apt-get update

policy=$(apt-cache policy kubeadm)
supported_versions="1.6 1.7 1.8"
unstable_versions="alpha|beta|rc"
skipped=""
missing=""

for release in $(curl -s https://api.github.com/repos/kubernetes/kubernetes/releases | jq -r '.[].name'); do
	minor=$(echo $release | cut -f1,2 -d'.')
	if [[ $release =~ $unstable_versions ]]; then
		# alpha, beta, rc releases should be ignored
		continue
	fi
	if [[ $supported_versions != *"${minor#v}"* ]]; then
		# release we don't care about (e.g. older releases)
		skipped="$skipped $release"
		continue
	fi
    	if [[ $policy != *"${release#v}"* ]]; then
		# release we care about but has missing debs
		missing="$missing $release"
	fi
done

if [[ ! -z "$skipped" ]]; then
	echo "Skipped these versions:"
	echo "$skipped"
fi

if [[ ! -z "$missing" ]]; then
        echo "These versions do not have matching debs:"
        echo "$missing"
        exit 1
fi

