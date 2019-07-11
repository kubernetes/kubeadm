Forked from Kind v0.4.0 release branch to expose some functionality currently not
accessible using kind as a library.

Having direct control on kubeadm config and on loadbalancer config is a specific necessity
for kinder, because kinder supports containerd and docker as a container runtime installed inside the
nodes, while kinder supports only containerd.

As a consiquence of the different behaviour at create node time, it was necessary to reimplement in kinder the kubeadm-config loadbalancer config actions, that currently are not accessible using kind as a library.
