# Create a base box for kubeadm-playground

Define a cluster api spec with a single machine. Then create the
machine installing only prerequisites:

```bash
kubeadm-playground start prerequisites
```

Once the machine is created connect with `kubeadm-playground ssh` and pre-pull images
for your target kubernetes versions:

```bash
sudo docker pull k8s.gcr.io/pause-amd64:3.1

sudo docker pull k8s.gcr.io/etcd-amd64:3.1.12

sudo docker pull k8s.gcr.io/kube-apiserver-amd64:v1.10.2 
sudo docker pull k8s.gcr.io/kube-scheduler-amd64:v1.10.2 
sudo docker pull k8s.gcr.io/kube-controller-manager-amd64:v1.10.2 
sudo docker pull k8s.gcr.io/kube-proxy-amd64:v1.10.2 

sudo docker pull k8s.gcr.io/k8s-dns-dnsmasq-nanny-amd64:1.14.8
sudo docker pull k8s.gcr.io/k8s-dns-sidecar-amd64:1.14.8
sudo docker pull k8s.gcr.io/k8s-dns-kube-dns-amd64:1.14.8
```

Pre-pull images for the external vip installation:
```bash
sudo docker pull ctracey/ucarp
```

Then, pre-pull images for your network provider:
```bash
sudo docker pull weaveworks/weave-npc:2.3.0
sudo docker pull weaveworks/weave-kube:2.3.0
```

or

```bash
sudo docker pull quay.io/coreos/flannel:v0.10.0
```

or 

```bash
sudo docker pull quay.io/calico/node:v3.1.1
sudo docker pull quay.io/calico/cni:v3.1.1
sudo docker pull quay.io/calico/kube-controllers:v3.1.1
sudo docker pull quay.io/coreos/etcd:v3.1.10
```

Finally clean up disk space for reducing the size of the base image
```bash
sudo apt-get clean
sudo dd if=/dev/zero of=/EMPTY bs=1M
sudo rm -f /EMPTY
cat /dev/null > ~/.bash_history && history -c && exit
```

# package and deploy the VM
```bash
vagrant package --output kubeadm-playground.box

vagrant box remove kubeadm-playground-1-10-2/ubuntu-xenial64 | true
vagrant box add kubeadm-playground-1-10-2/ubuntu-xenial64 kubeadm-playground.box
```

# cleanup

```bash
kubeadm-playground delete

rm kubeadm-playground.box
```
