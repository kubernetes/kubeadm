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
import collections

def validate_integer(arg):
    try:
        int(arg)
    except ValueError:
        raise ValueError("invalid integer '%s'" % (arg)) 

CIDR_PATTERN = "^(?=\d+\.\d+\.\d+\.\d+($|\/))(([1-9]?\d|1\d\d|2[0-4]\d|25[0-5])\.?){4}(\/([0-9]|[1-2][0-9]|3[0-2]))?$"

def validate_cidr(arg):
    if re.match(CIDR_PATTERN, arg) == None:
        raise ValueError("invalid CIDR '%s'" % (arg)) 

def validate_cidrs(*args):
    for arg in args:
        validate_cidr(arg)

def dict_merge(dct, merge_dct):
    """ Recursive dict merge. The ``merge_dct`` is merged into ``dct`` """
    for k, v in merge_dct.iteritems():
        if (k in dct and isinstance(dct[k], dict)
                and isinstance(merge_dct[k], collections.Mapping)):
            dict_merge(dct[k], merge_dct[k])
        else:
            dct[k] = merge_dct[k]
            
def print_H1(arg):
    """ print H1 title """
    print ("\nKUBEADM PLAYGROUND [%s] " % (arg)).ljust(80, "*")

def print_H2(arg):
    """ print H2 title """
    print "\n\033[1;32m%s:\033[0m" % (arg)
    print

def print_normal(*args):
    """ print normal text """
    for arg in args:
        print "\033[0;34m%s\033[0m" % (arg)

def print_warning(*args):
    """ print warning messages """
    print ''
    for arg in args:
        print "\033[1;33m%s\033[0m" % (arg)
    print ''

def print_error(arg):
    """ print error messages """
    print ''
    print "\033[0;31m%s\033[0m" % (arg)
    print ''

def print_code(*args):
    """ print code text """
    print
    print "\033[0;40m%s\033[0m" % (' '.ljust(120))
    prompt = '$'
    for arg in args:
        print "\033[0;40m  %s %s\033[0m" % (prompt, arg.ljust(116))
        prompt = ' ' if arg[-1]=="\\" else '$'
    print "\033[0;40m%s\033[0m" % (' '.ljust(120))
    print
