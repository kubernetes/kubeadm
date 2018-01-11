# Hacking on Kubeadm e2e tests

This is a guide for folks who want to hack on e2e tests. Contributions are welcome!

## Brief overview

* Actual tests are part of [kubernetes e2e test suite](https://github.com/kubernetes/test-infra/tree/master/kubetest)
* Environments are getting provisioned using [kubernetes-anywhere](https://github.com/kubernetes/kubernetes-anywhere) repository

## Set up Tools

**GOPATH**

This guide is using GOPATH, but you are welcome to switch it to any other directory

Set up gopath here:

https://golang.org/doc/code.html#GOPATH

**Install jsonnet**

```
git clone git@github.com:google/jsonnet.git
cd jsonnet
# if you don't have GOPATH set, use alternative location
cp jsonnet $GOPATH/bin
```

**Install JQ**

On Ubuntu/Debian:

`sudo apt-get -y install jq`

**Install Terraform**

Terraform version must be of 0.7.2 otherwise e2e tests wont work.

```
curl -o /tmp/terraform_0.7.2_linux_amd64.zip https://releases.hashicorp.com/terraform/0.7.2/terraform_0.7.2_linux_amd64.zip
cd $GOPATH/bin
unzip /tmp/terraform_0.7.2_linux_amd64.zip
rm -f /tmp/terraform_0.7.2_linux_amd64.zip
```

## Set up your GCE and 

**Install GCE SDK**

Read the guide here: https://cloud.google.com/sdk/downloads

**Clone Repos**

```
mkdir -p $GOPATH/src/github.com/kuberentes
cd $GOPATH/src/github.com/kuberentes

git clone git@github.com:kubernetes/kubernetes.git
git clone git@github.com:kubernetes/test-infra.git
go install github.com/kubernetes/test-infra/kubetest

# note that pipejakob is a temporary fix for the issue in kubernetes-anywhere
mkdir -p $GOPATH/src/github.com/pipejakob
cd $GOPATH/src/github.com/pipejakob
git clone github.com/pipejakob/kubernetes-anywhere.git
```

**Create bucket**

gcloud auth login
Go through the IAM setup described here:

https://github.com/kubernetes/kubernetes-anywhere/tree/master/phase1/gce


```bash
cd $GOPATH/src/github.com/pipejakob/kubernetes-anywhere
$ export PROJECT_ID=<my-project>
$ export PROJECT=<my-project>
$ export SERVICE_ACCOUNT="kubernetes-anywhere@${PROJECT_ID}.iam.gserviceaccount.com"
$ gcloud iam service-accounts create kubernetes-anywhere \
    --display-name kubernetes-anywhere
$ gcloud iam service-accounts keys create phase1/gce/account.json \
    --iam-account "${SERVICE_ACCOUNT}"
$ gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
    --member "serviceAccount:${SERVICE_ACCOUNT}" --role roles/editor
```

**Generate SSH Key**

ssh-keygen -t rsa -f ~/.ssh/google_compute_engine

## Launch e2e

```bash
cd $GOPATH/src/github.com/kubernetes/kubernetes
kubetest -v --deployment=kubernetes-anywhere --kubernetes-anywhere-path ${GOPATH}/src/github.com/pipejakob/kubernetes-anywhere --kubernetes-anywhere-phase2-provider kubeadm --kubernetes-anywhere-cluster my-e2e-test --up --test --down
```

## Troubleshooting

### Instance account failure

If you are seeing the error:

```
Error applying plan:

1 error(s) occurred:

* google_compute_instance_group_manager.my-e2e-test-node-group: The service account '<>@cloudservices.gserviceaccount.com' is not associated with the project.


Terraform does not automatically rollback in the face of errors.
Instead, your Terraform state file has been partially updated with
any resources that successfully completed. Please address the error
above and apply again to incrementally change your infrastructure.
```

Re-run `gcloud init` and set up your default region to `us-central1-b`

Have no idea why is it the case, but it works :/

### Failure to spin up weave

You need to make sure that your host local kubectl is the same version as the cluster you are testing, or you can get errors on the client:

```
unable to decode ".tmp/weave-net-cluster-role-binding.json": no kind "ClusterRoleBinding" is registered for version "rbac.authorization.k8s.io/v1beta1"
unable to decode ".tmp/weave-net-cluster-role.json": no kind "ClusterRole" is registered for version "rbac.authorization.k8s.io/v1beta1"
Makefile:57: recipe for target 'addons' failed
```
