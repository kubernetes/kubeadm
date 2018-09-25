## Tools for verifying Kubernetes packages are published

The `verify_packages_published.sh` script in this folder verifies that the necessary deb/rpms are published to official repositories.
This script is run periodically and results are available at https://k8s-testgrid.appspot.com/sig-cluster-lifecycle-all#periodic-kubernetes-e2e-packages-pushed

### Running

The script should be run in a Docker container like this:

```
docker run -it -v $(pwd):/test debian:stretch /test/tests/e2e/packages/verify_packages_published.sh
```
