## CHANGELOG for kubeadm releases:

#### Second release between v1.4 and v1.5: `v1.5.0-alpha.2.421+a6bea3d79b8bba`
 - Switch to the 10.96.0.0/12 subnet: [#35290](https://github.com/kubernetes/kubernetes/pull/35290)
 - Fix kubeadm on AWS by including /etc/ssl/certs in the controller-manager [#33681](https://github.com/kubernetes/kubernetes/pull/33681)
 - The API was refactored and is now componentconfig: [#33728](https://github.com/kubernetes/kubernetes/pull/33728), [#34147](https://github.com/kubernetes/kubernetes/pull/34147) and [#34555](https://github.com/kubernetes/kubernetes/pull/34555)
 - Allow kubeadm to get config options from a file: [#34501](https://github.com/kubernetes/kubernetes/pull/34501), [#34885](https://github.com/kubernetes/kubernetes/pull/34885) and [#34891](https://github.com/kubernetes/kubernetes/pull/34891)
 - Implement preflight checks: [#34341](https://github.com/kubernetes/kubernetes/pull/34341) and [#35843](https://github.com/kubernetes/kubernetes/pull/35843)
 - Using kubernetes v1.4.4 by default: [#34419](https://github.com/kubernetes/kubernetes/pull/34419) and [#35270](https://github.com/kubernetes/kubernetes/pull/35270)
 - Make api and discovery ports configurable and default to 6443: [#34719](https://github.com/kubernetes/kubernetes/pull/34719)
 - Implement kubeadm reset: [#34807](https://github.com/kubernetes/kubernetes/pull/34807)
 - Make kubeadm poll/wait for endpoints instead of directly fail when the master isn't available [#34703](https://github.com/kubernetes/kubernetes/pull/34703) and [#34718](https://github.com/kubernetes/kubernetes/pull/34718)
 - Allow empty directories in the directory preflight check: [#35632](https://github.com/kubernetes/kubernetes/pull/35632)
 - Started adding unit tests: [#35231](https://github.com/kubernetes/kubernetes/pull/35231), [#35326](https://github.com/kubernetes/kubernetes/pull/35326) and [#35332](https://github.com/kubernetes/kubernetes/pull/35332)
 - Various enhancements: [#35075](https://github.com/kubernetes/kubernetes/pull/35075), [#35111](https://github.com/kubernetes/kubernetes/pull/35111), [#35119](https://github.com/kubernetes/kubernetes/pull/35119), [#35124](https://github.com/kubernetes/kubernetes/pull/35124), [#35265](https://github.com/kubernetes/kubernetes/pull/35265) and [#35777](https://github.com/kubernetes/kubernetes/pull/35777)
 - Bug fixes: [#34352](https://github.com/kubernetes/kubernetes/pull/34352), [#34558](https://github.com/kubernetes/kubernetes/pull/34558), [#34573](https://github.com/kubernetes/kubernetes/pull/34573), [#34834](https://github.com/kubernetes/kubernetes/pull/34834), [#34607](https://github.com/kubernetes/kubernetes/pull/34607), [#34907](https://github.com/kubernetes/kubernetes/pull/34907) and [#35796](https://github.com/kubernetes/kubernetes/pull/35796)

#### Initial v1.4 release: `v1.5.0-alpha.0.1534+cf7301f16c0363`
