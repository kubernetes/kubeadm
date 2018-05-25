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

import re
import os
import sys
import yaml
import shutil
import subprocess

import kubernetes_utils

hack_lib_folder = os.path.dirname(os.path.abspath(__file__))
root_folder = os.path.abspath(os.path.join(hack_lib_folder, '../..'))
ansible_folder = os.path.join(root_folder, 'hack/ansible')
bin_folder = os.path.join(root_folder, 'bin')
tmp_folder = os.path.join(root_folder, 'tmp')

def import_kubeadm_from_build_output_path(builder, prefix):
    """ copies copies kubeadm build to vagrant and store vars override for future usages """

    # import kubeadm binary
    kubernetes_kubeadm_binary = os.path.join(kubernetes_utils.build_output_path(builder), 'kubeadm')

    if prefix != '':
        prefix = prefix + '_'

    vagrant_kubeadm_binary = os.path.join(bin_folder, "%skubeadm" % (prefix))

    if not os.path.exists(bin_folder):
        os.makedirs(bin_folder)

    shutil.copyfile(kubernetes_kubeadm_binary, vagrant_kubeadm_binary)

    # stores the var override
    extra_vars_override = { 
        'kubeadm': { 
            'binary': vagrant_kubeadm_binary.replace(bin_folder, '/vagrant/bin') 
        }
    }
    
    if not os.path.exists(tmp_folder):
        os.makedirs(tmp_folder)

    extra_vars_target = os.path.join(tmp_folder, 'extra_vars_override.yml')

    with open(extra_vars_target, 'w') as outfile:
        yaml.dump(extra_vars_override, outfile, default_flow_style=False)

    return extra_vars_override['kubeadm']['binary']

def get_status():
    """ Run `vagrant status` and parse the current vm state """
    node_state = {}

    output = check_vagrant(['status'])
    for i, line in enumerate(output.splitlines()):
        if i < 2:
            continue
        parts = re.split('\s+', line)
        if len(parts) == 3:
            node_state[parts[0]] = parts[1]
        elif len(parts) == 4:
            node_state[parts[0]] = " ".join(parts[1:3])
    
    return node_state

def exec_up(fallbackMode):
    """ Bring up the vm's with a `vagrant up` """

    fallbackOption = []
    if fallbackMode:
        fallbackOption = ['--provision-with fallback_bootstrap']
    run_vagrant(['up', '--parallel'])

def write_sshconfig():
    """ Run `vagrant ssh-config` to get ssh connection info and makes it available to ansible """
    output = subprocess.check_output(['vagrant', 'ssh-config'])

    if not os.path.exists(tmp_folder):
        os.makedirs(tmp_folder)
    
    ssh_config = os.path.join(tmp_folder, 'ssh_config')

    with open(ssh_config, 'w') as fh:
        fh.write(output)

def exec_halt():
    """ Halts vm's with a `vagrant halt` """
    run_vagrant(['halt'])

def exec_destroy():
    """ Destroy vm's with a `vagrant destroy` """
    run_vagrant(['destroy', '-f'])

def exec_ssh(machine):
    """ SSH into a machine using `vagrant ssh` """
    run_vagrant(['ssh', machine])

def check_vagrant(args):
    try:           
        return subprocess.check_output(['vagrant'] + args, cwd=root_folder)
    except Exception as e:
        raise RuntimeError('Error executing `vagrant ' + args[0] + '`: ' + repr(e))

def run_vagrant(args):
    try:           
        subprocess.call(['vagrant'] + args, cwd=root_folder)
    except Exception as e:
        raise RuntimeError('Error executing `vagrant ' + args[0] + '`: ' + repr(e))

def clean_working_folders():
    """ clean up temporary folders """
    import shutil
    for d in [bin_folder, tmp_folder]:
        if os.path.exists(d):
            shutil.rmtree(d)

