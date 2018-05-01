Vagrant.configure("2") do |config|
  config.vm.box = "bento/ubuntu-17.10"

  config.vm.provider "virtualbox" do |vb|
    vb.cpus   = "4"
    vb.memory = "4096"
  end

  config.vm.provision "shell", inline: <<-SHELL
    #!/bin/sh
    set -ex
    
    apt-get update && apt-get install -y apt-transport-https
    curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
    echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" > /etc/apt/sources.list.d/kubernetes.list
    apt-get update && apt-get install -y \
      docker.io=1.13.1-0ubuntu6 \
      kubelet=1.9.7-00 \
      kubectl=1.9.7-00 \
      kubeadm=1.9.7-00
    
    swapoff -a
    systemctl enable docker.service

    export KUBECONFIG=/etc/kubernetes/admin.conf
    echo "export KUBECONFIG=/etc/kubernetes/admin.conf" >> /etc/bash.bashrc
    # sudo -i
    # /vagrant/bin/594_kubeadm init
    # kubectl apply -f https://git.io/weave-kube-1.6
    # kubectl taint nodes --all node-role.kubernetes.io/master-
  SHELL
end
