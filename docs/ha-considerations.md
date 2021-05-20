# High Availability Considerations

This document contains a collection of community-provided considerations for setting up High Availability Kubernetes clusters. If something is incomplete, not clear or for additional information, please feel free to create a PR for a contribution. A good place for asking questions or making remarks is the `#kubeadm` channel on the [Kubernetes slack](https://slack.k8s.io/) where most of the contributors are usually active.

<!-- TOC -->

- [High Availability Considerations](#high-availability-considerations)
    - [Overview](#overview)
    - [Options for Software Load Balancing](#options-for-software-load-balancing)
    - [keepalived and haproxy](#keepalived-and-haproxy)
    - [kube-vip](#kube-vip)
    - [Bootstrap the cluster](#bootstrap-the-cluster)

## Overview

When setting up a production cluster, high availability (the cluster's ability to remain operational even if some control plane or worker nodes fail) is usually a requirement. For worker nodes, assuming that there are enough of them, this is part of the very cluster functionality. However redundancy of control plane nodes and `etcd` instances needs to be catered for when planning and setting up a cluster.

`kubeadm` supports setting up of multi control plane and multi `etcd` clusters (see [Creating Highly Available clusters with kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/high-availability/) for a step-by-step guide). Still there are some aspects to consider and set up which are not part of Kubernetes itself and hence not covered in the project documentation. This document provides some additional information and examples useful when planning and bootstrapping HA clusters with `kubeadm`.

## Options for Software Load Balancing

When setting up a cluster with more than one control plane, higher availability can be achieved by putting the API Server instances behind a load balancer and using the `--control-plane-endpoint` option when running `kubeadm init` for the new cluster to use it.

Of course, the load balancer itself should be highly available, too. This is usually achieved by adding redundancy to the load balancer. In order to do so, a cluster of hosts managing a virtual IP is set up with each host running an instance of the load balancer, so that always the load balancer on the host currently holding the vIP will be used while the others are on standby.

In some environments, like in data centers with dedicated load balancing components (provided e.g. by some cloud-providers), this functionality may already be available. If it is not, user-managed load balancing can be used. In that case some preparation is necessary before bootstrapping a cluster.

Since this is not part of Kubernetes or `kubeadm`, this must be taken care of separately. In the following sections, we give examples that have been working for some people while of course there are potentially dozens of other possible configurations.

## keepalived and haproxy

For providing load balancing from a virtual IP the combination [keepalived](https://www.keepalived.org) and [haproxy](https://www.haproxy.com) has been around for a long time and can be considered well-known and well-tested:
- The `keepalived` service provides a virtual IP managed by a configurable health check. Due to the way the virtual IP is implemented, all the hosts between which the virtual IP is negotiated need to be in the same IP subnet.
- The `haproxy` service can be configured for simple stream-based load balancing thus allowing TLS termination to be handled by the API Server instances behind it.

This combination can be run either as services on the operating system or as static pods on the control plane hosts. The service configuration is identical for both cases.

### keepalived configuration

The `keepalived` configuration consists of two files: the service configuration file and a health check script which will be called periodically to verify that the node holding the virtual IP is still operational.

The files are assumed to reside in a `/etc/keepalived` directory. Note that however some Linux distributions may keep them elsewhere. The following configuration has been successfully used with `keepalived` version 2.0.17:

```bash
! /etc/keepalived/keepalived.conf
! Configuration File for keepalived
global_defs {
    router_id LVS_DEVEL
}
vrrp_script check_apiserver {
  script "/etc/keepalived/check_apiserver.sh"
  interval 3
  weight -2
  fall 10
  rise 2
}

vrrp_instance VI_1 {
    state ${STATE}
    interface ${INTERFACE}
    virtual_router_id ${ROUTER_ID}
    priority ${PRIORITY}
    authentication {
        auth_type PASS
        auth_pass ${AUTH_PASS}
    }
    virtual_ipaddress {
        ${APISERVER_VIP}
    }
    track_script {
        check_apiserver
    }
}
```

There are some placeholders in `bash` variable style to fill in:
- `${STATE}` is `MASTER` for one and `BACKUP` for all other hosts, hence the virtual IP will initially be assigned to the `MASTER`.
- `${INTERFACE}` is the network interface taking part in the negotiation of the virtual IP, e.g. `eth0`.
- `${ROUTER_ID}` should be the same for all `keepalived` cluster hosts while unique amongst all clusters in the same subnet. Many distros pre-configure its value to `51`.
- `${PRIORITY}` should be higher on the control plane node than on the backups. Hence `101` and `100` respectively will suffice.
- `${AUTH_PASS}` should be the same for all `keepalived` cluster hosts, e.g. `42`
- `${APISERVER_VIP}` is the virtual IP address negotiated between the `keepalived` cluster hosts.

The above `keepalived` configuration uses a health check script `/etc/keepalived/check_apiserver.sh` responsible for making sure that on the node holding the virtual IP the API Server is available. This script could look like this:

```
#!/bin/sh

errorExit() {
    echo "*** $*" 1>&2
    exit 1
}

curl --silent --max-time 2 --insecure https://localhost:${APISERVER_DEST_PORT}/ -o /dev/null || errorExit "Error GET https://localhost:${APISERVER_DEST_PORT}/"
if ip addr | grep -q ${APISERVER_VIP}; then
    curl --silent --max-time 2 --insecure https://${APISERVER_VIP}:${APISERVER_DEST_PORT}/ -o /dev/null || errorExit "Error GET https://${APISERVER_VIP}:${APISERVER_DEST_PORT}/"
fi
```

There are some placeholders in `bash` variable style to fill in:
- `${APISERVER_VIP}` is the virtual IP address negotiated between the `keepalived` cluster hosts.
- `${APISERVER_DEST_PORT}` the port through which Kubernetes will talk to the API Server.

### haproxy configuration

The `haproxy` configuration consists of one file: the service configuration file which is assumed to reside in a `/etc/haproxy` directory. Note that however some Linux distributions may keep them elsewhere. The following configuration has been successfully used with `haproxy` version 2.1.4:

```bash
# /etc/haproxy/haproxy.cfg
#---------------------------------------------------------------------
# Global settings
#---------------------------------------------------------------------
global
    log /dev/log local0
    log /dev/log local1 notice
    daemon

#---------------------------------------------------------------------
# common defaults that all the 'listen' and 'backend' sections will
# use if not designated in their block
#---------------------------------------------------------------------
defaults
    mode                    http
    log                     global
    option                  httplog
    option                  dontlognull
    option http-server-close
    option forwardfor       except 127.0.0.0/8
    option                  redispatch
    retries                 1
    timeout http-request    10s
    timeout queue           20s
    timeout connect         5s
    timeout client          20s
    timeout server          20s
    timeout http-keep-alive 10s
    timeout check           10s

#---------------------------------------------------------------------
# apiserver frontend which proxys to the control plane nodes
#---------------------------------------------------------------------
frontend apiserver
    bind *:${APISERVER_DEST_PORT}
    mode tcp
    option tcplog
    default_backend apiserver

#---------------------------------------------------------------------
# round robin balancing for apiserver
#---------------------------------------------------------------------
backend apiserver
    option httpchk GET /healthz
    http-check expect status 200
    mode tcp
    option ssl-hello-chk
    balance     roundrobin
        server ${HOST1_ID} ${HOST1_ADDRESS}:${APISERVER_SRC_PORT} check
        # [...]
```
Again, there are some placeholders in `bash` variable style to expand:
- `${APISERVER_DEST_PORT}` the port through which Kubernetes will talk to the API Server.
- `${APISERVER_SRC_PORT}` the port used by the API Server instances
- `${HOST1_ID}` a symbolic name for the first load-balanced API Server host
- `${HOST1_ADDRESS}` a resolvable address (DNS name, IP address) for the first load-balanced API Server host
- additional `server` lines, one for each load-balanced API Server host

### Option 1: Run the services on the operating system

In order to run the two services on the operating system, the respective distribution's package manager can be used to install the software. This can make sense if they will be running on dedicated hosts not part of the Kubernetes cluster.

Having now installed the abovementioned configuration, the services can be enabled and started. On a recent RedHat-based system, `systemd` will be used for this:
```
# systemctl enable haproxy --now
# systemctl enable keepalived --now
```
With the services up, now the Kubernetes cluster can be bootstrapped using `kubeadm init` (see [below](#bootstrap-the-cluster)).

### Option 2: Run the services as static pods

If `keepalived` and `haproxy` will be running on the control plane nodes they can be configured to run as static pods. All that is necessary here is placing respective manifest files in the `/etc/kubernetes/manifests` directory before bootstrapping the cluster. During the bootstrap process, `kubelet` will bring the processes up, so that the cluster can use them while starting. This is an elegant solution, in particular with the setup described under [Stacked control plane and etcd nodes](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/high-availability/#stacked-control-plane-and-etcd-nodes).

For this setup, two manifest files need to be created in `/etc/kubernetes/manifests` (create the directory first).

The manifest for `keepalived`, `/etc/kubernetes/manifests/keepalived.yaml`:
```yaml
apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  name: keepalived
  namespace: kube-system
spec:
  containers:
  - image: osixia/keepalived:2.0.17
    name: keepalived
    resources: {}
    securityContext:
      capabilities:
        add:
        - NET_ADMIN
        - NET_BROADCAST
        - NET_RAW
    volumeMounts:
    - mountPath: /usr/local/etc/keepalived/keepalived.conf
      name: config
    - mountPath: /etc/keepalived/check_apiserver.sh
      name: check
  hostNetwork: true
  volumes:
  - hostPath:
      path: /etc/keepalived/keepalived.conf
    name: config
  - hostPath:
      path: /etc/keepalived/check_apiserver.sh
    name: check
status: {}
```

The manifest for `haproxy`, `/etc/kubernetes/manifests/haproxy.yaml`:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: haproxy
  namespace: kube-system
spec:
  containers:
  - image: haproxy:2.1.4
    name: haproxy
    livenessProbe:
      failureThreshold: 8
      httpGet:
        host: localhost
        path: /healthz
        port: ${APISERVER_DEST_PORT}
        scheme: HTTPS
    volumeMounts:
    - mountPath: /usr/local/etc/haproxy/haproxy.cfg
      name: haproxyconf
      readOnly: true
  hostNetwork: true
  volumes:
  - hostPath:
      path: /etc/haproxy/haproxy.cfg
      type: FileOrCreate
    name: haproxyconf
status: {}
```

Note that here again a placeholder needs to be filled in: `${APISERVER_DEST_PORT}` needs to hold the same value as in `/etc/haproxy/haproxy.cfg` (see above).

This combination has been successfully used with the versions used in the example. Other versions might work as well or may require changes to the configuration files.

With the services up, now the Kubernetes cluster can be bootstrapped using `kubeadm init` (see [below](#bootstrap-the-cluster)).

## kube-vip

As an alternative to the more "traditional" approach of `keepalived` and `haproxy`, [kube-vip](https://plndr.io/kube-vip/) implements both management of a virtual IP and load balancing in one service. Similar to option 2 above, `kube-vip` will be run as a static pod on the control plane nodes.

Like with `keepalived`, the hosts negotiating a virtual IP need to be in the same IP subnet. Similarly, like with `haproxy`, stream-based load-balancing allows TLS termination to be handled by the API Server instances behind it.

The configuration file `/etc/kube-vip/config.yaml` looks like this:
```yaml
localPeer:
  id: ${ID}
  address: ${IPADDR}
  port: 10000
remotePeers:
- id: ${PEER1_ID}
  address: ${PEER1_IPADDR}
  port: 10000
# [...]
vip: ${APISERVER_VIP}
gratuitousARP: true
singleNode: false
startAsLeader: ${IS_LEADER}
interface: ${INTERFACE}
loadBalancers:
- name: API Server Load Balancer
  type: tcp
  port: ${APISERVER_DEST_PORT}
  bindToVip: false
  backends:
  - port: ${APISERVER_SRC_PORT}
    address: ${HOST1_ADDRESS}
  # [...]
```

The `bash` style placeholders to expand are these:
- `${ID}` the current host's symbolic name
- `${IPADDR}` the current host's IP address
- `${PEER1_ID}` a symbolic name for the first vIP peer
- `${PEER1_IPADDR}` IP address for the first vIP peer
- entries (`id`, `address`, `port`) for additional vIP peers can follow
- `${APISERVER_VIP}` is the virtual IP address negotiated between the `kube-vip` cluster hosts.
- `${IS_LEADER}` is `true` for exactly one leader and `false` for the rest
- `${INTERFACE}` is the network interface taking part in the negotiation of the virtual IP, e.g. `eth0`.
- `${APISERVER_DEST_PORT}` the port through which Kubernetes will talk to the API Server.
- `${APISERVER_SRC_PORT}` the port used by the API Server instances
- `${HOST1_ADDRESS}` the first load-balanced API Server host's IP address
- entries (`port`, `address`) for additional load-balanced API Server hosts can follow

To have the service started with the cluster, now the manifest `kube-vip.yaml` needs to be placed in `/etc/kubernetes/manifests` (create the directory first). It can be generated using the `kube-vip` docker image:
```
# docker run -it --rm plndr/kube-vip:0.1.1 /kube-vip sample manifest \
    | sed "s|plndr/kube-vip:'|plndr/kube-vip:0.1.1'|" \
    | sudo tee /etc/kubernetes/manifests/kube-vip.yaml
```

The result, `/etc/kubernetes/manifests/kube-vip.yaml`, will look like this:
```yaml
apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  name: kube-vip
  namespace: kube-system
spec:
  containers:
  - command:
    - /kube-vip
    - start
    - -c
    - /vip.yaml
    image: 'plndr/kube-vip:0.1.1'
    name: kube-vip
    resources: {}
    securityContext:
      capabilities:
        add:
        - NET_ADMIN
        - SYS_TIME
    volumeMounts:
    - mountPath: /vip.yaml
      name: config
  hostNetwork: true
  volumes:
  - hostPath:
      path: /etc/kube-vip/config.yaml
    name: config
status: {}
```

With the services up, now the Kubernetes cluster can be bootstrapped using `kubeadm init` (see [below](#bootstrap-the-cluster)).

## Bootstrap the cluster

Now the actual cluster bootstrap as described in [Creating Highly Available clusters with kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/high-availability/) can take place.

Note that, if `${APISERVER_DEST_PORT}` has been configured to a value different from `6443` in the configuration above, `kubeadm init` needs to be told to use that port for the API Server. Assuming that in a new cluster port 8443 is used for the load-balanced API Server and a virtual IP with the DNS name `vip.mycluster.local`, an argument `--control-plane-endpoint` needs to be passed to `kubeadm` as follows:

```
# kubeadm init --control-plane-endpoint vip.mycluster.local:8443 [additional arguments ...]
```
