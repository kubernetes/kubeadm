# cluster api specification

Cluster API uses two objects for specifying the cluster topology, Cluster and MachineSets.

## kind: Cluster

Allows to specify some characteristics of the cluster, selecting among all the supported kubeadm options. 
You can define:

- The target kubernetes/kubeadm version
- How the controlplane will be deployed (static pods vs self hosting)
- Which type of certificate authority are you going to use (local vs external)
- Where your PKI will be stored (filesystem or secrets)
- Which type of DNS are you going to use (kubeDNS or coreDNS)
- Which type of pod network add-on are you going to use (weavenet, flannel, calico)
- Which type of kubelet config are you going to use (systemd dropIn files, dynamic Kubelet config)
- TODO: Which type of network are you using (Ipv4, Ipv6)
- TODO: Which type of environment are you simulating (with or without internet connection)
- TODO: Auditing
- ...

Most of the configuration should be done in the `spec.providerConfig.value` object; this object
can be merged eventually with values passed via the `--extra-vars` flag or the `` environment variable.

``` yaml
apiVersion: cluster.k8s.io/v1alpha1
kind: Cluster
metadata:
  name: kubeadm-test
spec:
  providerConfig:
    value:
      apiVersion: vagrantCluster/v1alpha1
      kind: vagrantCluster
      kubernetes:
        version:
          <string (optional, default 1.10.2)
          defines the version of kubeadm, kubelet, kubectl .deb, .rpm and of the version of the
          controlplane components to be installed.
          See advanced usage for explanation about possible variations to the default version behavior>
        controlplaneVersion:
          <string (optional, default empty)
          See advanced usage for explanation about possible variations to the default version behavior>
        controlplane:
          <staticPods|selfHosting (optional, default staticPods)
          defines how the controlplane will be deployed>
        certificateAuthority:
          <local|external (optional, default local)
          defines the type of certificate authority that will be used. local certificateAuthority are
          created by kubeadm, while external certificateAuthority should be provided before creating
          the cluster>
        pkiLocation:
          <filesystem|secrets (optional, default filesystem)
          defines the target location for the certificate authority>
        dnsAddon:
          <kubeDNS|coreDNS (optional, default kubeDNS)
          defines the target DNS addon of choice>
        kubeletConfig:
          <systemdDropIn|dynamicKubeletConfig (optional, default systemdDropIn)
          defines the target DNS addon of choice>
        cni:
          plugin:
            <weavenet|flannel|calico (optional, default weavenet)
            defines the target network add-on of choice>
      etcd:
        version: <string (optional, default v3.2.17), defines the version of etcd binaries to be
        installed - only external etcd - >
      kubeadm:
        apiVersion: <string (optional, default v1alpha1), defines API version to use for the
          kubeadm.conf file - >
        token: <string (optional, default abcdef.0123456789abcdef), defines the token string to use
          for kubeadm init and join - >
```

## kind: MachineSet

Allows you to define the target cluster topology e.g. add master nodes, add worker nodes, or create
an external etcd cluster.

You can define more than one MachineSet, assigning different roles to each set.

Supported roles are `Master`, `Node` and also `Etcd`. Please note that:

- The `Etcd` role should be used only when it is necessary to spin up an external etcd.
- Roles can be combined, e.g. for hosting the external etcd on machine with role `Master`.

```yaml
apiVersion: cluster.k8s.io/v1alpha1
kind: MachineSet
metadata:
  name: Masters
spec:
  replicas: 1
  template:
    metadata:
    spec:
      providerConfig:
        value:
          apiVersion: vagrantproviderconfig/v1alpha1
          kind: VagrantProviderConfig
          box:
            <string (required)
            defines the target vagrant box to use for this machine>
          cpus:
            <int (required)
            defines the number of cores to assign to this machine>
          memory:
            <int (required)
            defines the amount of memory to assign to this machine>
```

NB. the `version` object defined in cluster API is ignored! Use the `version` config value at
cluster level instead.