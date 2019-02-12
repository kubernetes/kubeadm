## Tools for verifying Kubernetes packages

The tests in this folder run periodically and their status is available here:
https://k8s-testgrid.appspot.com/sig-cluster-lifecycle-all

### Package publishing

The `verify_packages_published.sh` script verifies that the necessary DEB/RPM packages are published to official repositories.

Running locally:
```
docker run -it -v $(pwd):/test debian:stretch /test/tests/e2e/packages/verify_packages_published.sh
```

### Package installation

The `verify_packages_install_deb.sh` verifies that the DEB packages from our release repositories and CI build
can be installed and uninstalled successfully.

Running locally:
```
docker run -it -v $(pwd):/test debian:stretch /test/tests/e2e/packages/verify_packages_install_deb.sh
```
