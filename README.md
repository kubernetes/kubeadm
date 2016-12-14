# Checkpoint

## Description

`checkpoint` is a utility application which will manage "checkpoints" of pods scheduled to the local node.

The purpose of a pod checkpoint is to ensure that existing local pod state can be recovered in the absence of an api-server.

The kubelet will already attempt to ensure that local pods will continue to run in the absence of an API server (based on restartPolicy). However, if all local runtime state has been lost (for example after a reboot), the checkpoint utility will ensure the local state can be recovered until an api-server is contacted.

This is accomplished by managing checkpoints as static pod manifests:

- When the checkpointer sees that a "parent pod" (a pod which should be checkpointed), is succesfully running, the checkpointer will save a local copy of the manifest.

- If the parent pod is detected as no longer running, the checkpointer will "activate" the checkpoint manifest. It will allow the checkpoint to continue running until the parent-pod is restarted on the local node, or it is able to contact an api-server to determine that the parent pod is no longer scheduled to this node.

## Use

Any pod which contains the `checkpointer.alpha.coreos.com/checkpoint=true` annotation will be considered a viable "parent pod" which should be checkpointed.
The parent pod cannot itself be a static pod, and is not a checkpoint itself.

Checkpoints are denoted by the `checkpointer.alpha.coreos.com/checkpoint-of` annotation. This annotation will point to the parent of this checkpoint by pod name.

For example the pod:

```
apiVersion: v1
kind: Pod
metadata:
  name: kube-apiserver
  namespace: kube-system
  annotations:
    checkpointer.alpha.coreos.com/checkpoint=true
```

Will generate a checkpointed pod as:

```
apiVersion: v1
kind: Pod
metadata:
  name: kube-apiserver
  namespace: kube-system
  annotations:
    checkpointer.alpha.coreos.com/checkpoint-of=kube-apiserver
```

## Implementation Notes:

### Asset Locations

- Inactive checkpoint manifests: /srv/kubernetes/manifests
- Active checkpoint manifests: /etc/kubernetes/manifests
- Checkpointed secrets: /etc/kubernetes/checkpoint-secrets
- Config Maps: TODO

### Pod Manifest Sanitization

Parts of the pod manifest will be scrubbed prior to being saved as checkpoints. This is to ensure that the pod does not interfere with the parent object, and is managed in isolation.

 - All labels and non-checkpoint related annotations will be removed
 - Service account details are removed
 - Secrets are downloaded from the apiserver and converted to hostMounts
 - When support is added for ConfigMaps, these will also be stored locally and converted to hostMounts
 - Pod status is cleared

### Secret Storage

Secrets are stored using a path of:

```
/etc/kubernets/checkpoint-secrets/<namespace>/<pod-name>/<secret-name>
```

