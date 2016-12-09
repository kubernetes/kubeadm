## CHANGELOG for kubeadm releases:

#### Third release that deploys v1.5 clusters by default:
 - Incorporated the system verification that validates the node before trying to run kubeadm check into the preflight checks [#36334](https://github.com/kubernetes/kubernetes/pull/36334)
 - The control plane version automatically defaults to the latest stable version of kubernetes [#37222](https://github.com/kubernetes/kubernetes/pull/37222)
 - Use `etcd-3.0.14` as the default etcd version [#35921](https://github.com/kubernetes/kubernetes/pull/35921)
 - Added a new command: `kubeadm token generate` [#35144](https://github.com/kubernetes/kubernetes/pull/35144)
 - `kubeadm reset` drains the node it's run on and defaults to removing the node from the API Server as well [#37831](https://github.com/kubernetes/kubernetes/pull/37831)
 - Updated the DNS addon to the v1.5 version [#37835](https://github.com/kubernetes/kubernetes/pull/37835)
 - `kubeadm` now auto-starts the `kubelet` systemd service if inactive [#37568](https://github.com/kubernetes/kubernetes/pull/37568)
 - Made the console output of `kubeadm` much cleaner and user-friendly [#37568](https://github.com/kubernetes/kubernetes/pull/37568)
 - `kubectl logs` and `kubectl exec` can now be used with `kubeadm` clusters [#37568](https://github.com/kubernetes/kubernetes/pull/37568)
 - Changed SELinux directive from `unconfined_t` to `spc_t` on some control plane Pods [#37327](https://github.com/kubernetes/kubernetes/pull/37327)
 - Removes the content of `/etc/cni/net.d` when running `kubeadm reset` [#37831](https://github.com/kubernetes/kubernetes/pull/37831)
 - Mount `/etc/pki` into the control plane containers if present [#36373](https://github.com/kubernetes/kubernetes/pull/36373)
 - The image repository prefix can be changed via the environment variable `KUBE_REPO_PREFIX` as a _temporary solution_ [#35948](https://github.com/kubernetes/kubernetes/pull/35948)
 - The logging level for components are now `--v=2` [#35933](https://github.com/kubernetes/kubernetes/pull/35933)
 - Better preflight checks in general [#35972](https://github.com/kubernetes/kubernetes/pull/35972), [#37084](https://github.com/kubernetes/kubernetes/pull/37084), [#37498](https://github.com/kubernetes/kubernetes/pull/37498), [#37524](https://github.com/kubernetes/kubernetes/pull/37524), [#36083](https://github.com/kubernetes/kubernetes/pull/36083)
 - New unit tests: [#36106](https://github.com/kubernetes/kubernetes/pull/36106), [#36263](https://github.com/kubernetes/kubernetes/pull/36263)
 - Other improvements: [#36040](https://github.com/kubernetes/kubernetes/pull/36040), [#36625](https://github.com/kubernetes/kubernetes/pull/36625), [#36474](https://github.com/kubernetes/kubernetes/pull/36474), [#37494](https://github.com/kubernetes/kubernetes/pull/37494), [#36025](https://github.com/kubernetes/kubernetes/pull/36025)

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
