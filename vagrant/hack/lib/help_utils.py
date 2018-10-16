# -*- coding: utf-8 -*-

# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import utils
import cluster_api
import kubeadm_utils

def print_help(cluster, machines, topic):
    """ print help for the selected topic """

    hi = HelpInfo(cluster, machines)

    if topic == 'external-etcd':
        print_help_external_etcd(hi)
    elif topic == 'external-ca':
        print_help_external_ca(hi)
    elif topic == 'external-vip':
        print_help_external_vip(hi)
    elif topic == 'external-ca':
        print_help_external_ca(hi)
    elif topic == 'kubeadm-init':
        print_help_kubeadm_init(hi)
    elif topic == 'kubectl-apply-network':
        print_help_kubectl_apply_network(hi)
    elif topic == 'kubeadm-join':
        print_help_kubeadm_join(hi)
    else:
        utils.print_warning('No local help is defined for this topic.'
                            'Please refer to https://github.com/kubernetes/kubeadm/blob/master/vagrant/README.md') 

def print_help_external_etcd(hi):
    """ help for external etcd installation """

    utils.print_H1 ('How to install an external Etcd')
    utils.print_normal ('By default external etcd will be automatically installed by `kubeadm-playground create` ',
                "in case your cluster has machines with role '%s'." % (cluster_api.ROLE_ETCD))
    if len(hi.etcds) == 0:
        print
        utils.print_warning ('The kubeadm playground currently does not require an external etcd.',
                             'Please change your cluster api specification')

    utils.print_H2 ('Assisted mode')
    utils.print_normal ("* only if automatic installation was disabled during create")
    utils.print_code ("kubeadm-playground exec external-etcd")

    utils.print_H2 ('Manual mode')
    utils.print_normal ('- Install etcd on following machines:')
    for m in hi.etcds:
        utils.print_normal ("   - %s" % (m.name))
    print
    utils.print_normal ("- Ensure that etcd endpoint and eventually etcd TLS certificates are set in '/etc/kubernetes/kubeadm.conf' on %s" % (hi.bootstrapMaster.name))
    print

def print_help_external_ca(hi):
    """ help for external ca installation """

    utils.print_H1 ('How to setup an external Certificate Authority')    
    utils.print_normal ('By default external CA will be automatically installed by `kubeadm-playground create` ',
                'in case your cluster is configured with `certificateAuthority: external`')
    if hi.cluster.certificateAuthority!=kubeadm_utils.CERTIFICATEAUTHORITY_TYPE_EXTERNAL:
        print
        utils.print_warning ('The kubeadm playground currently does not require an external ca.',
                                'Please change your cluster api specification')

    utils.print_H2 ('Assisted mode')
    utils.print_normal ("* only if automatic installation was disabled during create")
    utils.print_code ("kubeadm-playground exec external-ca")

    utils.print_H2 ('Manual mode')
    ssh_to (hi.bootstrapMaster.name)
    utils.print_normal ('- Create an external CA by executing:')
    utils.print_code ("sudo %s alpha phase certs all --config /etc/kubernetes/kubeadm.conf" % (hi.kubeadm_binary),
                'sudo rm /etcd/kubernetes/pki/ca.key',
                'sudo rm /etcd/kubernetes/pki/front-proxy-ca.key')  
    print

def print_help_external_vip(hi):
    """ help for external vip installation """
    
    utils.print_H1 ('How to setup an external Vip/load balancer')
    if len(hi.etcds) == 0:
        print
        utils.print_warning ('The kubeadm playground currently does not require an external vip.',
                             'Please change your cluster api specification')

    utils.print_normal ('By default external etcd will be automatically installed by `kubeadm-playground create` ',
                "in case your cluster has more then one machines with role '%s'." % (cluster_api.ROLE_MASTER))

    utils.print_H2 ('Assisted mode')
    utils.print_normal ("* only if automatic installation was disabled during create")

    utils.print_H2 ('Assisted mode')
    utils.print_normal ('By default external vip/load balancer will be automatically installed by `kubeadm-playground create` ',
                'in case your cluster is configured with more than one machine with role `Master`',
                ''
                "The vip address will be %s (%s) and will balance following api server end points:" % (hi.kubernetes_vip_fqdn, hi.kubernetes_vip_ip))
    for m in hi.masters:
        utils.print_normal ("- https://%s:6443" % (m.ip))

    print
    utils.print_normal ('If automatic installation of external vip was disabled during create, it can be invoked afterwards with:')

    utils.print_code ("kubeadm-playground exec external-vip")

    utils.print_H2 ('Manual mode')
    utils.print_normal ('- Create an external VIP/load balancer similar to what described above.')
    utils.print_normal ("- Ensure that the VIP address is set in '/etc/kubernetes/kubeadm.conf' on %s." % (hi.bootstrapMaster.name))
    print

def print_help_kubeadm_init(hi):
    utils.print_H1 ('How to execute kubeadm init')
    
    utils.print_H2 ('Assisted mode')
    utils.print_code ("kubeadm-playground exec kubeadm-init")

    utils.print_H2 ('Manual mode')
    ssh_to (hi.bootstrapMaster.name)
    utils.print_normal ('- Initialize the kubernetes master node')
    utils.print_code ("sudo %s init --config /etc/kubernetes/kubeadm.conf" % (hi.kubeadm_binary))
 
def print_help_kubectl_apply_network(hi):
    utils.print_H1 ("How to install %s network addon" % (hi.networkAddon))
    
    utils.print_H2 ('Assisted mode')
    utils.print_code ("kubeadm-playground exec kubectl-apply-network")

    utils.print_H2 ('Manual mode')
    ssh_to (hi.bootstrapMaster.name)

    if hi.kubernetes_cni_sysconf:
        utils.print_normal ("- Make required changes for %s CNI plugin to work:" % (hi.networkAddon))  
        utils.print_code ('sudo sysctl net.bridge.bridge-nf-call-iptables=1')
    utils.print_normal ("- Install %s CNI plugin:" % (hi.networkAddon))
    utils.print_code ('kubectl apply \\',
            "     -f %s" % (hi.kubernetes_cni_manifest_url))

def print_help_kubeadm_join(hi):
    utils.print_H1 ("How to execute kubeadm join")
    if len(hi.nodes) == 0:
        print
        utils.print_warning ('The kubeadm playground currently does not have worker nodes.',
                             'Please change your cluster api specification')

    utils.print_H2 ('Assisted mode')
    utils.print_code ("kubeadm-playground exec kubeadm-join")

    utils.print_H2 ('Manual mode')
    utils.print_normal ("Repeat following steps for all the machines with role '%s'" % (cluster_api.ROLE_NODE))
    if hi.kubernetes_cni_sysconf:
        utils.print_normal ("- Make required changes for %s CNI plugin to work:" % (hi.networkAddon))  
        utils.print_code ('sudo sysctl net.bridge.bridge-nf-call-iptables=1')
    utils.print_normal ('- Join the worker node')
    utils.print_code ("sudo %s join %s:6443 --token %s \\" % (hi.kubeadm_binary, hi.controlplaneEndpoint, hi.kubeadm_token),
                 "         --discovery-token-unsafe-skip-ca-verification")

class HelpInfo:
    """ Utility class for generating help """
    
    def __init__(self, cluster, machines):
        
        self.cluster  = cluster
        self.machines = machines
        self.masters  = [m for m in machines if cluster_api.ROLE_MASTER in m.roles]
        self.nodes    = [m for m in machines if cluster_api.ROLE_NODE in m.roles]
        self.etcds    = [m for m in machines if cluster_api.ROLE_ETCD in m.roles]
        self.bootstrapMaster = self.masters[0]

        self.kubernetes_vip_fqdn            = cluster.extra_vars['kubernetes']['vip']['fqdn']
        self.kubernetes_vip_ip              = cluster.extra_vars['kubernetes']['vip']['ip']
        if len(self.masters)>1:
            self.controlplaneEndpoint        = self.kubernetes_vip_fqdn
        else:
            self.controlplaneEndpoint        = self.bootstrapMaster.ip 
        self.networkAddon                   = cluster.networkAddon
        self.kubernetes_cni_manifest_url    = cluster.extra_vars['kubernetes']['cni'][self.networkAddon]['manifestUrl'].replace("\n", "\\n")
        self.kubernetes_cni_sysconf         = kubeadm_utils.network_addon_require_sysconf(self.networkAddon) 
        self.kubeadm_token                  = cluster.extra_vars['kubeadm']['token']
        self.kubeadm_binary                 = cluster.extra_vars['kubeadm']['binary']

def ssh_to(machine):    
    utils.print_normal ('From the guest machine:')
    utils.print_code ("kubeadm-playground ssh %s" % (machine))
    utils.print_normal ('After connecting:')
    print
