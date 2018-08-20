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

import os
import sys
import glob
import yaml

import utils
import kubeadm_utils
import vagrant_utils
import kubernetes_utils

ROLE_MASTER  = 'Master'      
ROLE_NODE    = 'Node' 
ROLE_ETCD    = 'Etcd'

def validate_roles(args):
    valid = [ROLE_MASTER, ROLE_NODE, ROLE_ETCD]
    for arg in args:
        if arg not in valid:
            raise ValueError("invalid machine set role '%s'. Valid roles are %s" % (arg, ', '.join(valid))) 

class VagrantCluster:
    """ represent a VagrantCluster, as defined in the cluster API  """

    def __init__(self):
 
        self.name = ""
        self.version = ""
        self.controlplane = ""
        self.certificateAuthority = ""
        self.pkiLocation = ""
        self.dnsAddon = ""
        self.dnsDomain = ""                 
        self.networkAddon = ""              
        self.serviceSubnet = ""             
        self.podSubnet = ""
        self.kubeletConfig = ""
        self.extra_vars = {}                # the whole providerConfig.value is used as extra_vars for ansible 
        self.highavailability = False       # nb derived from machine sets (Master machines > 1)
        self.externalEtcd = False           # nb derived from machine sets (Etcd machines > 0)

class VagrantMachineSet:
    """ represent a VagrantMachineSet, as defined in the cluster API  """

    def __init__(self):
        self.name       = ""
        self.replicas   = 0
        self.box        = ""
        self.cpus       = 0
        self.memory     = 0
        self.roles      = []

class VagrantMachine:
    """ represent a VagrantMachine creating from a VagrantMachineSet specification """

    def __init__(self):
        self.name       = ""
        self.hostname   = ""
        self.box        = ""
        self.ip         = ""
        self.cpus       = 0
        self.memory     = 0
        self.roles      = []

def parse(folder):
    """ parses the cluster api definition in a folder """
    if not os.path.isabs(folder):
        folder = os.path.join(vagrant_utils.root_folder, folder)

    cluster = None
    machineSets = []

    for f in glob.glob(folder):
        if os.path.isfile(f):
            try:           
                data = yaml.load(open(f))

                if data["kind"] == "Cluster":
                    cluster = get_cluster(data)
                elif data["kind"] == "MachineSet":
                    machineSets.append(get_machineSet(data))
            except Exception as e:
                raise RuntimeError('Error parsing cluster api specification in `' + repr(f) + '`: ' + repr(e))

    # fails fast if incomplete configurations are detected
    if cluster == None: 
        raise ValueError("Invalid cluster api specification in %s. Cluster object not defined" % (spec))
    
    if len(machineSets) == 0: 
        raise ValueError("Invalid cluster api specification in %s. MachineSets objects not defined" % (spec))

    return cluster, machineSets

def get_cluster(data):
    """ get the cluster definition from a cluster api object """

    c = VagrantCluster()

    c.name                       = getx(data, "metadata.name", validator=kubernetes_utils.validate_rfc123_label)
    c.version                    = getx(data, "spec.providerConfig.value.kubernetes.version") #TODO: validate version
    c.controlplane               = getx(data, "spec.providerConfig.value.kubernetes.controlplane", default=kubeadm_utils.CONTROLPLANE_TYPE_STATICPODS, validator=kubeadm_utils.validate_controlplane_type)
    c.certificateAuthority       = getx(data, "spec.providerConfig.value.kubernetes.certificateAuthority", default=kubeadm_utils.CERTIFICATEAUTHORITY_TYPE_LOCAL, validator=kubeadm_utils.validate_certificateAuthority_type)
    c.pkiLocation                = getx(data, "spec.providerConfig.value.kubernetes.pkiLocation", default=kubeadm_utils.CERTIFICATEAUTHORITY_LOCATION_FILESYSTEM, validator=kubeadm_utils.validate_certificateAuthority_location)
    c.dnsAddon                   = getx(data, "spec.providerConfig.value.kubernetes.dnsAddon", default=kubeadm_utils.DNS_ADDON_KUBE_DNS, validator=kubeadm_utils.validate_dns_addon)
    c.dnsDomain                  = getx(data, "spec.clusterNetwork.serviceDomain", default=kubeadm_utils.DEFAULT_DNS_DOMAIN , validator=kubernetes_utils.validate_rfc123_subdomain)
    c.networkAddon               = getx(data, "spec.providerConfig.value.kubernetes.cni.plugin", default=kubeadm_utils.NETWORK_ADDON_WEAVENET, validator=kubeadm_utils.validate_network_addon)
    (serviceSubnet, podSubnet) = kubeadm_utils.network_addon_default_subnet(c.networkAddon)
    c.serviceSubnet              = getx(data, "spec.clusterNetwork.services.cidrblocks", default=serviceSubnet, validator=utils.validate_cidr)
    c.podSubnet                  = getx(data, "spec.clusterNetwork.pods.cidrblocks", default=podSubnet, validator=utils.validate_cidrs)
    c.kubeletConfig              = getx(data, "spec.providerConfig.value.kubernetes.kubeletConfig", default=kubeadm_utils.KUBELET_CONFIG_TYPE_SYSTEMDDROPIN, validator=kubeadm_utils.validate_kubelet_config_type)

    # uses the providerConfig as an input for ansible
    c.extra_vars                 = getx(data, "spec.providerConfig.value")
    
    # values defined in cluster API attributes, are replicate under extra_vars for simplifying ansible integration
    if c.dnsDomain != kubeadm_utils.DEFAULT_DNS_DOMAIN:
        c.extra_vars['kubernetes']['dnsDomain'] = c.dnsDomain

    if c.serviceSubnet[0] != None:
        c.extra_vars['kubernetes']['serviceSubnet'] = c.serviceSubnet[0]

    if c.podSubnet[0] != None:
        c.extra_vars['kubernetes']['podSubnet']     = c.podSubnet[0]

    # fails fast if not consistent configurations are detected
    if c.pkiLocation == kubeadm_utils.CERTIFICATEAUTHORITY_LOCATION_SECRETS and c.controlplane != kubeadm_utils.CONTROLPLANE_TYPE_SELFHOSTING: 
        raise ValueError("Invalid cluster definition. PKILocation can be secrets only if controlplane is self hosted")

    return c

def get_machineSet(data):
    """ get the machine set definition from a cluster api object """

    s = VagrantMachineSet()

    s.name        = getx(data, "metadata.name", validator=kubernetes_utils.validate_rfc123_label)
    s.replicas    = getx(data, "spec.replicas", default=1, validator=utils.validate_integer)
    s.box         = getx(data, "spec.template.spec.providerConfig.value.box", default="ubuntu/xenial64") # TODO:validator
    s.cpus        = getx(data, "spec.template.spec.providerConfig.value.cpus", default=2, validator=utils.validate_integer)
    s.memory      = getx(data, "spec.template.spec.providerConfig.value.memory", default=2048, validator=utils.validate_integer)
    s.roles       = getx(data, "spec.template.spec.roles", validator=validate_roles)
   
    return s

def getx(data, keys, default=None, validator=None):
    """ extended get of an attribute of the cluster api with, recoursion (deep get), defaulting & validation """

    for key in keys.split('.'):
        try:
            data = data[key]
        except KeyError:
            if default != None:
                return default
            else:
                raise KeyError("invalid cluster api definition. Key '%s' does not exist" % (keys)) 
    
    if validator != None:
        validator(data)

    return data

def get_machines(cluster, machineSets):
    """ transform a cluster / a list of machineSets into a set of machines to be created.
        the list of machine is also stored into `tmp/machines.yaml` for vagrant execution """

    machines = []
    j = 1
    for s in machineSets:
        for i in range(1, s.replicas + 1):
            m = VagrantMachine()
            m.name              = "%s-%s" % (cluster.name, s.name) if s.replicas == 1 else "%s-%s%s" % (cluster.name, s.name, j)
            m.hostname          = "%s-%s.local" % (cluster.name, m.name)
            m.box               = s.box
            m.ip                = "10.10.10.1%s" % (j)
            m.cpus              = s.cpus
            m.memory            = s.memory
            m.roles             = s.roles
            machines.append(m)
            j += 1

    masters = [m for m in machines if ROLE_MASTER in m.roles]
    if len(masters) == 0:
        raise ValueError("Invalid cluster definition. At least one Master machine is required")
    elif len(masters) > 1:
        cluster.highavailability = True
        if sum([1 for m in machines if ROLE_ETCD in m.roles])==0:
            raise ValueError("Invalid cluster definition. Multi masters requires external etcd.")
        if cluster.pkiLocation == kubeadm_utils.CERTIFICATEAUTHORITY_LOCATION_SECRETS:
            raise ValueError("Invalid cluster definition. Multi masters does not support certificates in secrets yet.")

    etcds = [m for m in machines if ROLE_ETCD in m.roles]
    if len(etcds) > 0:
        cluster.externalEtcd = True

    if not os.path.exists(vagrant_utils.tmp_folder):
        os.makedirs(vagrant_utils.tmp_folder)

    machines_target = os.path.join(vagrant_utils.tmp_folder, 'machines.yml')
    
    with open(machines_target, 'w') as outfile:
        yaml.dump(machines, outfile, default_flow_style=False)

    return machines

def fallbackSettings(cluster, machineSets):
    """ Fixes cluster and machineSets options in case of ansible not available,
        ensuring compatibility with the fallback_boostrap script defined in the Vagrant file """ 

    for s in machineSets:
        s.box = "bento/ubuntu-17.10" # base box tested with the boostrap script
