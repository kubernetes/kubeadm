## Tools for verifying kubeadm required images and manifest lists

The `verify_manifest_lists.sh` script in this folder verifies that the necessary multi-architecture Docker Schema 2
manifest lists are pushed to the GCR bucket that kubeadm uses by default.
The script is just a wrapper for a Go application.

It is run periodically and results are available at https://k8s-testgrid.appspot.com/sig-cluster-lifecycle-all#periodic-e2e-verify-manifest-lists

### Running

If you have `go`, `curl` and `gcc` installed you can run the script like so:

```
./verify_manifest_lists.sh
```

To run it in a docker container use this:

```
docker run -it -v $(pwd):/test debian:stretch /test/tests/e2e/manifests/verify_manifest_lists.sh
```
