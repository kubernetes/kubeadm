#!/bin/bash

# Install dependencies
apt-get update && apt-get install -y apt-transport-https curl jq gnupg2 yum

# Set up Kubernetes packages via apt
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
cat <<EOF >/etc/apt/sources.list.d/kubernetes.list
deb http://apt.kubernetes.io/ kubernetes-xenial main
EOF
apt-get update

# Set up Kubernetes packages via yum
mkdir -p /etc/yum.repos.d
cat <<EOF > /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64
enabled=1
gpgcheck=0
repo_gpgcheck=0
EOF
yum update -y

# Detect missing packages
deb_policy=$(apt-cache policy kubeadm)
rpm_policy=$(yum --showduplicates list kubeadm)
num_supported_versions=3
# supported_versions will be dynamically populated
supported_versions=""
latest_minor_version=$(curl -sSL dl.k8s.io/release/stable.txt | cut -f2 -d'.')
unstable_versions="alpha|beta|rc"
skipped=""
missing=""
available=""

for (( i = 0; i < ${num_supported_versions}; i++ )); do
	supported_versions="1.$((latest_minor_version-$i)) ${supported_versions}"
done

for release in $(curl -s https://api.github.com/repos/kubernetes/kubernetes/releases | jq -r '.[].name'); do
	minor=$(echo $release | cut -f1,2 -d'.')
	if [[ $release =~ $unstable_versions ]]; then
		# alpha, beta, rc releases should be ignored
		echo "Unstable version $release ignored"
	elif [[ $supported_versions != *"${minor#v}"* ]]; then
		# release we don't care about (e.g. older releases)
		skipped="$skipped $release"
	else
		if [[ $deb_policy != *"${release#v}"* ]]; then
			# release we care about but has missing debs
			missing="$missing deb:$release"
		else
			# All good, the expected deb package is available
			available="$available deb:$release"
		fi
		if [[ $rpm_policy != *"${release#v}"* ]]; then
			# release we care about but has missing rpms
			missing="$missing rpm:$release"
		else
			# All good, the expected rpm package is available
			available="$available rpm:$release"
		fi
	fi
done

if [[ ! -z "$skipped" ]]; then
	echo "Skipped these versions because they aren't supported:"
	echo "$skipped"
fi

if [[ ! -z "$available" ]]; then
	echo "These expected packages were found:"
	echo "$available"
fi

if [[ ! -z "$missing" ]]; then
	echo "ERROR: These versions do not have matching packages:"
	echo "$missing"
	exit 1
else
	echo ""
	echo "TESTS PASSED!! All necessary packages are pushed!"
	echo ""
fi
