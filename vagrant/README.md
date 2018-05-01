#### pre-requisite tools
`vagrant, bazel, golang`

# vagrant kubeadm testing
This [Vagrantfile](./Vagrantfile) provisions a 4cpu/4GB Ubuntu 17.10 vm for the purpose of developing on kubeadm.   

Recent packages of kubernetes components will be installed and a compatible Docker daemon is started and running.  
The bashrc is configured by default to use `/etc/kubernetes/admin.conf`.  
An upstream version of kubeadm is installed for reference usage.  

[copy_kubeadm_bin.sh](./copy_kubeadm_bin.sh) takes your freshly built kubeadm binary, prefixes it with an issue
number and copies it into `./bin`, so that it may be used from `/vagrant/bin` inside the VM for testing.  
This is useful for comparing behavior and determining compatibility of binairies from builds.

## example
#### build two versions of kubeadm:
```shell
cd ~/go/src/k8s.io/kubernetes

git checkout feature/kubeadm_594-etcd_tls
bazel test //cmd/kubeadm/...
bazel build //cmd/kubeadm --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64
issue=594 ~/go/src/k8s.io/kubeadm/vagrant/copy_kubeadm_bin.sh

git checkout feature/kubeadm_710-etcd-ca
bazel test //cmd/kubeadm/...
bazel build //cmd/kubeadm --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64
issue=710 ~/go/src/k8s.io/kubeadm/vagrant/copy_kubeadm_bin.sh
```
#### experiment with the two builds on the vagrant:
```shell
cd ~/go/src/k8s.io/kubeadm/vagrant

vagrant up
vagrant ssh
  sudo /vagrant/bin/594_kubeadm init
  sudo /vagrant/bin/594_kubeadm reset
  sudo /vagrant/bin/710_kubeadm init
```
