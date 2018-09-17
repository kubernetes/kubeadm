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

DEFAULT_DNS_DOMAIN = "cluster.local"

# How the controlplane will be deployed ?
CONTROLPLANE_TYPE_STATICPODS  = 'staticPods'
CONTROLPLANE_TYPE_SELFHOSTING = 'selfHosting'

def validate_controlplane_type(arg):
    valid = [CONTROLPLANE_TYPE_STATICPODS, CONTROLPLANE_TYPE_SELFHOSTING]
    if arg not in valid:
        raise ValueError("invalid controlplane type '%s'. Valid types are %s" % (arg, ', '.join(valid)))

# Which type of certificate authority are you going to use ?
CERTIFICATEAUTHORITY_TYPE_LOCAL       = 'local'
CERTIFICATEAUTHORITY_TYPE_EXTERNAL    = 'external'

def validate_certificateAuthority_type(arg):
    valid = [CERTIFICATEAUTHORITY_TYPE_LOCAL, CERTIFICATEAUTHORITY_TYPE_EXTERNAL]
    if arg not in valid:
        raise ValueError("invalid certificateauthority type '%s'. Valid types are %s" % (arg, ', '.join(valid)))

 # Where you PKI will be stored?
CERTIFICATEAUTHORITY_LOCATION_FILESYSTEM = 'filesystem'
CERTIFICATEAUTHORITY_LOCATION_SECRETS = 'secrets'

def validate_certificateAuthority_location(arg):
    valid = [CERTIFICATEAUTHORITY_LOCATION_FILESYSTEM, CERTIFICATEAUTHORITY_LOCATION_SECRETS]
    if arg not in valid:
        raise ValueError("invalid certificateauthority location '%s'. Valid types are %s" % (arg, ', '.join(valid)))

# Which type of DNS you will use?
DNS_ADDON_KUBE_DNS     = 'kubeDNS'
DNS_ADDON_CORE_DNS     = 'coreDNS'

def validate_dns_addon(arg):
    valid = [DNS_ADDON_KUBE_DNS, DNS_ADDON_CORE_DNS]
    if arg not in valid:
        raise ValueError("invalid DNS add-on '%s'. Valid add-ons are %s" % (arg, ', '.join(valid)))

# Which type of pod network add-on are you using?
NETWORK_ADDON_WEAVENET    = 'weavenet'
NETWORK_ADDON_FLANNEL     = 'flannel'
NETWORK_ADDON_CALICO      = 'calico'

def validate_network_addon(arg):
    valid = [NETWORK_ADDON_WEAVENET, NETWORK_ADDON_FLANNEL, NETWORK_ADDON_CALICO]
    if arg not in valid:
        raise ValueError("invalid DNS add-on '%s'. Valid add-ons are %s" % (arg, ', '.join(valid)))

def network_addon_default_subnet(addon):
    if addon == NETWORK_ADDON_WEAVENET:
        return [None], [None]
    elif addon == NETWORK_ADDON_FLANNEL:
        return [None], ["10.244.0.0/16"]
    elif addon == NETWORK_ADDON_CALICO:
        return [None], ["192.168.0.0/16"]

def network_addon_require_sysconf(addon):
    if addon in [NETWORK_ADDON_WEAVENET, NETWORK_ADDON_FLANNEL]:
        return True
    elif addon == NETWORK_ADDON_CALICO:
        return False

# Which type of kubelet config are you using?
KUBELET_CONFIG_TYPE_SYSTEMDDROPIN           = 'systemdDropIn'
KUBELET_CONFIG_TYPE_DYNAMICKUBELETCONFIG    = 'dynamicKubeletConfig' # technically this is SystemdDropIn + DynamicKubeletConfig

def validate_kubelet_config_type(arg):
    valid = [KUBELET_CONFIG_TYPE_SYSTEMDDROPIN, KUBELET_CONFIG_TYPE_DYNAMICKUBELETCONFIG]
    if arg not in valid:
        raise ValueError("invalid kubelet config type '%s'. Valid types are %s" % (arg, ', '.join(valid)))
