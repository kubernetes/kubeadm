# Role: Kubernetes

Install Kubernetes binaries and makes settings for making things working on Vagrant:

- sets kubelet --node-ip
- create a static route for service ip range (pod ip range will be managed by the network add-on)

Please note that the binary release of kubeadm is always installed to get the drop-in file include in packaging;
however the above kubeadm binary is not used by the kubeadm role if a different kubeadm binary is injected for testing.