import re
from ansible.errors import AnsibleFilterError

class FilterModule(object):

    def filters(self):
        return {
            'kube_platform_version': self.kube_platform_version,
            'deepGet': self.deepGet
        }

    def kube_platform_version(self, version, platform):
        match = re.match('(\d+\.\d+.\d+)$', version)
        if match:
            version = "%s-00" % (version)

        match = re.match('(\d+\.\d+.\d+)\-(\d+)', version)
        if not match:
            raise Exception("Version '%s' does not appear to be a "
                            "kubernetes version." % version)
        sub = match.groups(1)[1]
        if len(sub) == 1:
            if platform.lower() == "debian":
                return "%s-%s" % (match.groups(1)[0], '{:02d}'.format(sub))
            else:
                return version
        if len(sub) == 2:
            if platform.lower() == "redhat":
                return "%s-%s" % (match.groups(1)[0], int(sub))
            else:
                return version

        raise Exception("Could not parse kubernetes version")

    def deepGet(self, d, *ks, **kwargs):
        for k in ks:
            if k not in d:
                if 'default' in kwargs:
                    return kwargs.get('default')
                else:
                    raise AnsibleFilterError("attribute %s is not defined" % '.'.join(ks))
            d = d[k]
        return d