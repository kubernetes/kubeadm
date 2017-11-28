#!/bin/bash
yum update -y
cat <<EOF > /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF
yum update -y

yum install -y wget
wget https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 -O jq
chmod +x jq
mv jq /usr/local/bin

policy=$(yum --showduplicates list kubeadm)
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

