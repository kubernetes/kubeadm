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
import stat
import shutil
import subprocess

import kubernetes_utils

hack_lib_folder = os.path.dirname(os.path.abspath(__file__))
root_folder = os.path.abspath(os.path.join(hack_lib_folder, '../..'))
ansible_folder = os.path.join(root_folder, 'hack/ansible')
bin_folder = os.path.join(root_folder, 'bin')
packages_folder = os.path.join(bin_folder, 'packages')
tmp_folder = os.path.join(root_folder, 'tmp')

def import_kubeadm_binary(binary, builder, prefix):
    """ copies kubeadm binary to vagrant and store vars override for future usages """

    if binary != None:
        source_kubeadm_binary = binary
    else:
        source_kubeadm_binary = os.path.join(kubernetes_utils.build_output_path(builder), 'kubeadm')

    if prefix == None:
        prefix = ''
    else:
        prefix = prefix + '_'

    vagrant_kubeadm_binary = os.path.join(bin_folder, "%skubeadm" % (prefix))

    if not os.path.exists(bin_folder):
        os.makedirs(bin_folder)

    shutil.copyfile(source_kubeadm_binary, vagrant_kubeadm_binary)

    st = os.stat(vagrant_kubeadm_binary)
    os.chmod(vagrant_kubeadm_binary, st.st_mode | stat.S_IEXEC)

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

def import_packages(packages):
    """ copies packages to vagrant """

    if not os.path.exists(packages_folder):
        os.makedirs(packages_folder)

    for f in os.listdir(packages):
        filename, file_extension = os.path.splitext(f)
        if f in ['kubelet', 'kubectl']: # if supported binary files
            pass
        elif file_extension == ".deb" and filename in ['kubeadm', 'kubectl', 'kubelet',  'kubernetes-cni']: # if supported deb files
            pass
        elif file_extension == ".tar" and not filename.endswith("-internal-layer"): # if tar file - excluding internal layers
            pass
        else:
            continue # ignore the file

        s = os.path.join(packages, f)
        if os.path.isfile(s):
            d = os.path.join(packages_folder, f)
            shutil.copy2(s, d)

            st = os.stat(d)
            os.chmod(d, st.st_mode | stat.S_IRWXU)


def get_status():
    """ Run `vagrant status` and parse the current vm state """
    node_state = {}

    output = check_vagrant(['status'])
    for i, line in enumerate(output.splitlines()):
        if i < 2:
            continue
        if line == "":
            break
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
    output = check_vagrant(['ssh-config'])

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

