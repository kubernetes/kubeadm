# Checkpoint

`checkpoint` is a small utility that will convert an API server pod manifest
into a static manifest which can be stored on disk. The pod manifest is
obtained from the kubelets read only API.

This is useful for single-master clusters. A single-master cluster with this
checkpointing is resilient to certain failure scenarios, such as a system
reboot, or complete loss of the API server.

The tool waits until it detects an API server is no longer running.
In that case, it will move the static pod manifest created earlier into the
directory specified as the kubelets config directory. Once the tool detects our
normal API server has started up again, it will move the static manifest out of
the config directory, causing the kubelet to stop the checkpointed API server,
enabling the self-hosted API server to take over once more.
