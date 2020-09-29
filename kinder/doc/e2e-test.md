# Run E2E tests

You can use kinder as a proxy for running Kubernetes/Kubeadm e2e test suites, and use
those suites to validate your cluster.

## E2E (Kubernetes)

E2E Kubernetes is a rich set of test aimed at verifying the proper functioning of a Kubernetes cluster.

By default Kinder selects a subset of test the corresponds to the "Conformance" as defined in the
Kubernetes [test grid](https://git.k8s.io/test-infra/testgrid/conformance).

```bash
kinder test e2e
```

> The test are run on the cluster defined in the current kubeconfig file

Main flags supported by the command are:

- `--kube-root` for setting the folder where the kubernetes sources are stored
- `--conformance` as a shortcut for instructing the ginkgo test suite run only conformance tests
- `--parallel` as a shortcut for instructing the ginkgo to run test in parallel

See [Kinder reference](reference.md) for more options

## E2E kubeadm

Similarly to E2E Kubernetes, there is a suite of tests aimed at checking that kubeadm has created
and properly configured all the ConfigMap, Secrets, RBAC Roles and RoleBinding required for the
proper functioning of future calls to `kubeadm join` or `kubeadm upgrade`.

This can be achieved by a simple

```bash
kinder test e2e-kubeadm
```

> The test are run on the cluster defined in the current kubeconfig file

Main flags supported by the command are:

- `--kube-root` for setting the folder where the kubernetes sources are stored
- `--single-node` as a shortcut for instructing the ginkgo test suite to skip test labeled with [multi-node]
- `--automatic-copy-certs` as a shortcut for instructing the ginkgo test suite to skip test labeled with [copy-certs]

See [Kinder reference](reference.md) for more options
