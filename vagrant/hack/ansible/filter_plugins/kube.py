import re
from ansible.errors import AnsibleFilterError

_REGEX = re.compile(
        r"^(?:v?)(?P<major>(?:0|[1-9][0-9]*))\.(?P<minor>(?:0|[1-9][0-9]*))\.(?P<patch>(?:0|[1-9][0-9]*))(\-(?P<prerelease>(?:0|[1-9A-Za-z-][0-9A-Za-z-]*)(\.(?:0|[1-9A-Za-z-][0-9A-Za-z-]*))*))?(\+(?P<build>[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*))?$", re.VERBOSE)

def parse(version):
    """Parse version to major, minor, patch, pre-release, build parts. """
    match = _REGEX.match(version)
    if match is None:
        raise ValueError('%s is not valid SemVer string' % version)

    version_parts = match.groupdict()

    version_parts['major'] = int(version_parts['major'])
    version_parts['minor'] = int(version_parts['minor'])
    version_parts['patch'] = int(version_parts['patch'])

    return version_parts

def _compare_by_keys( version, other):
    
    d1 = parse(version)
    d2 = parse(other)
  
    for key in ['major', 'minor', 'patch']:
        v = cmp(d1.get(key), d2.get(key))
        if v:
            return v

    rc1, rc2 = d1.get('prerelease'), d2.get('prerelease')
    rccmp = _nat_cmp(rc1, rc2)

    if not rccmp:
        return 0
    if not rc1:
        return 1
    elif not rc2:
        return -1

    return rccmp
    
def _nat_cmp(a, b):
    def convert(text):
        return int(text) if re.match('^[0-9]+$', text) else text

    def split_key(key):
        return [convert(c) for c in key.split('.')]

    def cmp_prerelease_tag(a, b):
        if isinstance(a, int) and isinstance(b, int):
            return cmp(a, b)
        elif isinstance(a, int):
            return -1
        elif isinstance(b, int):
            return 1
        else:
            return cmp(a, b)

    a, b = a or '', b or ''
    a_parts, b_parts = split_key(a), split_key(b)
    for sub_a, sub_b in zip(a_parts, b_parts):
        cmp_result = cmp_prerelease_tag(sub_a, sub_b)
        if cmp_result != 0:
            return cmp_result
    else:
        return cmp(len(a), len(b))
        
class FilterModule(object):

    def filters(self):
        return {
            'kube_platform_version': self.kube_platform_version,
            'deepGet': self.deepGet,
            'semverlt': self.semverlt,
            'semverle': self.semverle,
            'semvergt': self.semvergt,
            'semverge': self.semverge
        }

    def semverlt(self, version, other):
        return _compare_by_keys(version, other) < 0

    def semverle(self, version, other):
        return _compare_by_keys(version, other) <= 0

    def semvergt(self, version, other):
        return _compare_by_keys(version, other) > 0

    def semverge(self, version, other):
        return _compare_by_keys(version, other) >= 0

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