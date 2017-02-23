package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_5"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/runtime"
)

const (
	nodeNameEnv = "NODE_NAME"

	// We must use both the :10255/pods and :10250/runningpods/ endpoints, because /pods endpoint could have stale data.
	// The /pods endpoint will only show the last cached status which has successfully been written to an apiserver.
	// However, if there is no apiserver, we may get stale state (e.g. saying pod is running, when it really is not).
	kubeletAPIPodsURL        = "http://127.0.0.1:10255/pods"
	kubeletAPIRunningPodsURL = "https://127.0.0.1:10250/runningpods/"

	activeCheckpointPath   = "/etc/kubernetes/manifests"
	inactiveCheckpointPath = "/srv/kubernetes/manifests"
	checkpointSecretPath   = "/etc/kubernetes/checkpoint-secrets"
	kubeconfigPath         = "/etc/kubernetes/kubeconfig"

	shouldCheckpointAnnotation = "checkpointer.alpha.coreos.com/checkpoint"    // = "true"
	checkpointParentAnnotation = "checkpointer.alpha.coreos.com/checkpoint-of" // = "podName"
	podSourceAnnotation        = "kubernetes.io/config.source"

	shouldCheckpoint = "true"
	podSourceFile    = "file"
)

//TODO(aaron): The checkpointer should know how to GC itself because it runs as a static pod.

func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()
	defer glog.Flush()

	nodeName := os.Getenv(nodeNameEnv)
	if nodeName == "" {
		glog.Fatalf("Missing required environment variable: %s", nodeNameEnv)
	}

	glog.Infof("Starting checkpointer for node: %s", nodeName)
	run(newClient(), nodeName)
}

func run(client clientset.Interface, nodeName string) {
	for {
		time.Sleep(3 * time.Second)

		localParentPods, err := getLocalParentPods()
		if err != nil {
			// If we can't determine local state from kubelet api, we shouldn't make any decisions about checkpoints.
			glog.Errorf("Failed to retrive pod list from kubelet api: %v", err)
			continue
		}

		localRunningPods, err := getLocalRunningPods()
		if err != nil {
			glog.Errorf("Failed to retrieve running pods from kubelet api: %v", err)
			continue
		}

		createCheckpointsForValidParents(client, localParentPods)

		// Try to get scheduled pods from the apiserver.
		// These will be used to GC checkpoints for parents no longer scheduled to this node.
		// A return value of nil is assumed to be "could not contact apiserver"
		// TODO(aaron): only check this every 30 seconds or so
		apiParentPods := getAPIParentPods(client, nodeName)

		// Get on disk copies of (in)active checkpoints
		//TODO(aaron): Could be racy to load from disk each time, but much easier than trying to keep in-memory state in sync.
		activeCheckpoints := getFileCheckpoints(activeCheckpointPath)
		inactiveCheckpoints := getFileCheckpoints(inactiveCheckpointPath)

		start, stop, remove := process(localRunningPods, localParentPods, apiParentPods, activeCheckpoints, inactiveCheckpoints)
		handleRemove(remove)
		handleStop(stop)
		handleStart(start)
	}
}

// process() makes decisions on which checkpoints need to be started, stopped, or removed.
// It makes this decision based on inspecting the states from kubelet, apiserver, active/inactive checkpoints.
//
// - localRunningPods: running pods retrieved from kubelet api. Minimal amount of info (no podStatus) as it is extracted from container runtime.
// - localParentPods: pod state from kubelet api for all "to be checkpointed" pods - podStatus may be stale (only as recent as last apiserver contact)
// - apiParentPods: pod state from the api server for all "to be checkpointed" pods
// - activeCheckpoints: checkpoint pod manifests which are currently active & in the static pod manifest
// - inactiveCheckpoints: checkpoint pod manifest which are stored in an inactive directory, but are ready to be activated
//
// The return values are checkpoints which should be started or stopped, and checkpoints which need to be removed altogether.
// The removal of a checkpoint means its parent is no longer scheduled to this node, and we need to GC active / inactive
// checkpoints as well as any secrets / configMaps which are no longer necessary.
func process(localRunningPods, localParentPods, apiParentPods, activeCheckpoints, inactiveCheckpoints map[string]*v1.Pod) (start, stop, remove []string) {

	// We can only make some GC decisions if we've successfully contacted an apiserver.
	// When apiParentPods == nil, that means we were not able to get an updated list of pods.
	removeMap := make(map[string]struct{})
	if apiParentPods != nil {

		// Scan for inacive checkpoints we should GC
		for id := range inactiveCheckpoints {
			// If the inactive checkpoint still has a parent pod, do nothing.
			// This means the kubelet thinks it should still be running, which has the same scheduling info that we do --
			// so we won't make any decisions about its checkpoint.
			// TODO(aaron): This is a safety check, and may not be necessary -- question is do we trust that the api state we received
			//              is accurate -- and that we should ignore our local state (or assume it could be inaccurate). For example,
			//              local kubelet pod state will be innacurate in the case that we can't contact apiserver (kubelet only keeps
			//              cached responses from api) -- however, we're assuming we've been able to contact api, so this likely is moot.
			if _, ok := localParentPods[id]; ok {
				glog.V(4).Infof("API GC: skipping inactive checkpoint %s", id)
				continue
			}

			// If the inactive checkpoint does not have a parent in the api-server, we must assume it should no longer be running on this node.
			// NOTE: It's possible that a replacement for this pod has not been rescheduled elsewhere, but that's not something we can base our decision on.
			//       For example, if a single scheduler is running, and the node is drained, the scheduler pod will be deleted and there will be no replacement.
			//       However, we don't know this, and as far as the checkpointer is concerned - that pod is no longer scheduled to this node.
			if _, ok := apiParentPods[id]; !ok {
				glog.V(4).Infof("API GC: should remove inactive checkpoint %s", id)
				removeMap[id] = struct{}{}
				delete(inactiveCheckpoints, id)
			}
		}

		// Scan active checkpoints we should GC
		for id := range activeCheckpoints {
			// If the active checkpoint does not have a parent in the api-server, we must assume it should no longer be running on this node.
			if _, ok := apiParentPods[id]; !ok {
				glog.V(4).Infof("API GC: should remove active checkpoint %s", id)
				removeMap[id] = struct{}{}
				delete(activeCheckpoints, id)
			}
		}
	}

	// Can make decisions about starting/stopping checkpoints just with local state.

	// If there is an inactive checkpoint, and no parent is running, start the checkpoint
	for id := range inactiveCheckpoints {
		if _, ok := localRunningPods[id]; !ok {
			glog.V(4).Infof("Should start checkpoint %s", id)
			start = append(start, id)
		}
	}

	// If there is an active checkpoint and a running pod, stop the active checkpoint
	// The parent may not be in a running state, but the kubelet is trying to start it so we should get out of the way.
	for id := range activeCheckpoints {
		if _, ok := localRunningPods[id]; ok {
			glog.V(4).Infof("Should stop checkpoint %s", id)
			stop = append(stop, id)
		}
	}

	// De-duped checkpoints to remove. If we decide to GC a checkpoint, we will clean up both inactive/active.
	for k := range removeMap {
		remove = append(remove, k)
	}

	return start, stop, remove
}

// createCheckpointsForValidParents will iterate through pods which are candidates for checkpointing, then:
// - checkpoint any remote assets they need (e.g. secrets)
// - sanitize their podSpec, removing unnecessary information
// - store the manifest on disk in an "inactive" checkpoint location
//TODO(aaron): Add support for checkpointing configMaps
func createCheckpointsForValidParents(client clientset.Interface, pods map[string]*v1.Pod) {
	for _, pod := range pods {
		// This merely check that the last kubelet pod state thinks this pod was running. It's possible that
		// state is actually stale (only as recent as last successful contact with api-server). However, this
		// does contain the full podSpec -- so we can still attempt to checkpoint with this "last known good state".
		//
		// We do not use the `localPodRunning` state, because while the runtime may think the pod/containers are running -
		// they may actually be in a failing state - and we've not successfully sent that podStatus to any api-server.
		if !isRunning(pod) {
			continue
		}
		id := PodFullName(pod)

		cp, err := copyPod(pod)
		if err != nil {
			glog.Errorf("Failed to create checkpoint pod copy for %s: %v", id, err)
			continue
		}

		cp, err = checkpointSecretVolumes(client, cp)
		if err != nil {
			//TODO(aaron): This can end up spamming logs at times when api-server is unavailable. To reduce spam
			//             we could only log error if api-server can't be contacted and existing secret doesn't exist.
			glog.Errorf("Failed to checkpoint secrets for pod %s: %v", id, err)
			continue
		}
		cp, err = sanitizeCheckpointPod(cp)
		if err != nil {
			glog.Errorf("Failed to sanitize manifest for %s: %v", id, err)
			continue
		}
		if err := writeCheckpointManifest(cp); err != nil {
			glog.Errorf("Failed to write checkpoint for %s: %v", id, err)
		}
	}
}

// writeCheckpointManifest will save the pod to the inactive checkpoint location if it doesn't already exist.
func writeCheckpointManifest(pod *v1.Pod) error {
	b, err := json.Marshal(pod)
	if err != nil {
		return err
	}
	path := filepath.Join(inactiveCheckpointPath, pod.Namespace+"-"+pod.Name+".json")
	oldb, err := ioutil.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if bytes.Compare(oldb, b) == 0 {
		glog.V(4).Infof("Checkpoint manifest for %s already exists. Skipping", PodFullName(pod))
		return nil
	}
	glog.Infof("Checkpointing manifest for %s", PodFullName(pod))
	return writeAndAtomicRename(path, b, 0644)
}

func sanitizeCheckpointPod(cp *v1.Pod) (*v1.Pod, error) {
	// Clear ObjectMeta except for name/namespace
	// NOTE(aaron): If we want to keep labels, we need to add a new label so the static pod
	//              will not be adopted by higher-level parent (e.g. daemonset/deployment).
	//              Otherwise you end up in situations where parent tries deleting mirror pods.
	cp.ObjectMeta = v1.ObjectMeta{
		Name:        cp.Name,
		Namespace:   cp.Namespace,
		Annotations: make(map[string]string),
	}

	// Track this checkpoint's parent pod
	cp.Annotations[checkpointParentAnnotation] = cp.Name

	// Remove Service Account
	cp.Spec.ServiceAccountName = ""
	cp.Spec.DeprecatedServiceAccount = ""

	// Clear pod status
	cp.Status.Reset()

	return cp, nil
}

// getFileCheckpoints will retrieve all checkpoint manifests from a given filepath.
func getFileCheckpoints(path string) map[string]*v1.Pod {
	checkpoints := make(map[string]*v1.Pod)

	fi, err := ioutil.ReadDir(path)
	if err != nil {
		glog.Fatalf("Failed to read checkpoint manifest path: %v", err)
	}

	for _, f := range fi {
		manifest := filepath.Join(path, f.Name())
		b, err := ioutil.ReadFile(manifest)
		if err != nil {
			glog.Errorf("Error reading manifest: %v", err)
			continue
		}

		cp := &v1.Pod{}
		if err := runtime.DecodeInto(api.Codecs.UniversalDecoder(), b, cp); err != nil {
			glog.Errorf("Error unmarshalling manifest from %s: %v", filepath.Join(path, f.Name()), err)
			continue
		}

		if isCheckpoint(cp) {
			if _, ok := checkpoints[PodFullName(cp)]; ok { // sanity check
				glog.Warningf("Found multiple checkpoint pods in %s with same id: %s", path, PodFullName(cp))
			}
			checkpoints[PodFullName(cp)] = cp
		}
	}
	return checkpoints
}

// getAPIParentPods will retrieve all pods from apiserver that are parents & should be checkpointed
func getAPIParentPods(client clientset.Interface, nodeName string) map[string]*v1.Pod {
	opts := v1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(api.PodHostField, nodeName).String(),
	}

	podList, err := client.Core().Pods(api.NamespaceAll).List(opts)
	if err != nil {
		glog.Warningf("Unable to contact APIServer, skipping garbage collection: %v", err)
		return nil
	}
	return podListToParentPods(podList)
}

// getAPIParentPods will retrieve all pods from kubelet api that are parents & should be checkpointed
func getLocalParentPods() (map[string]*v1.Pod, error) {
	resp, err := http.Get(kubeletAPIPodsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to contact kubelet pod api: %v", err)
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var podList v1.PodList
	if err := json.Unmarshal(b, &podList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal podlist: %v", err)
	}
	return podListToParentPods(&podList), nil
}

// Transports should be re-used and not created as needed: https://golang.org/pkg/net/http/#Transport
var insecureTransport *http.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}

// getLocalRunningPods uses the /runningpods/ kubelet api to retrieve the local container runtime pod state
func getLocalRunningPods() (map[string]*v1.Pod, error) {
	// TODO(aaron): The kubelet api is currently secured by a self-signed cert. We should update this to actually verify at some point
	client := &http.Client{Transport: insecureTransport}
	resp, err := client.Get(kubeletAPIRunningPodsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to contact kubelet pod api: %v", err)
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var podList v1.PodList
	if err := json.Unmarshal(b, &podList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal podlist: %v", err)
	}
	return podListToMap(&podList, filterNone), nil
}

// checkpointSecretVolumes ensures that all pod secrets are checkpointed locally, then converts the secret volume to a hostpath.
func checkpointSecretVolumes(client clientset.Interface, pod *v1.Pod) (*v1.Pod, error) {
	for i := range pod.Spec.Volumes {
		v := &pod.Spec.Volumes[i]
		if v.Secret == nil {
			continue
		}

		path, err := checkpointSecret(client, pod.Namespace, pod.Name, v.Secret.SecretName)
		if err != nil {
			return nil, fmt.Errorf("failed to checkpoint secret for pod %s/%s: %v", pod.Namespace, pod.Name, err)
		}

		v.HostPath = &v1.HostPathVolumeSource{Path: path}
		v.Secret = nil

	}
	return pod, nil
}

// checkpointSecret will locally store secret data.
// The path to the secret data becomes: checkpointSecretPath/namespace/podname/secretName/secret.file
// Where each "secret.file" is a key from the secret.Data field.
func checkpointSecret(client clientset.Interface, namespace, podName, secretName string) (string, error) {
	secret, err := client.Core().Secrets(namespace).Get(secretName)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret %s/%s: %v", namespace, secretName, err)
	}

	basePath := secretPath(namespace, podName, secretName)
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return "", fmt.Errorf("failed to create secret checkpoint path %s: %v", basePath, err)
	}
	// TODO(aaron): No need to store if already exists
	for f, d := range secret.Data {
		if err := writeAndAtomicRename(filepath.Join(basePath, f), d, 0600); err != nil {
			return "", fmt.Errorf("failed to write secret %s: %v", secret.Name, err)
		}
	}
	return basePath, nil
}

func handleRemove(remove []string) {
	for _, id := range remove {
		// Remove inactive checkpoints
		glog.Infof("Removing checkpoint of: %s", id)
		p := PodFullNameToInactiveCheckpointPath(id)
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			glog.Errorf("Failed to remove inactive checkpoint %s: %v", p, err)
			continue
		}
		// Remove active checkpoints
		p = PodFullNameToActiveCheckpointPath(id)
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			glog.Errorf("Failed to remove active checkpoint %s: %v", p, err)
			continue
		}
		// Remove Secrets
		p = PodFullNameToSecretPath(id)
		if err := os.RemoveAll(p); err != nil {
			glog.Errorf("Failed to remove pod secrets from %s: %s", p, err)
		}
		// TODO(aaron): Remove configMaps when supported
	}
}

func handleStop(stop []string) {
	for _, id := range stop {
		glog.Infof("Stopping active checkpoint: %s", id)
		p := PodFullNameToActiveCheckpointPath(id)
		if err := os.Remove(p); err != nil {
			if os.IsNotExist(err) { // Sanity check (it's fine - just want to surface this if it's occurring)
				glog.Warningf("Attempted to remove active checkpoint, but manifest no longer exists: %s", p)
			} else {
				glog.Errorf("Failed to stop active checkpoint %s: %v", p, err)
			}
		}
	}
}

func handleStart(start []string) {
	for _, id := range start {
		src := PodFullNameToInactiveCheckpointPath(id)
		data, err := ioutil.ReadFile(src)
		if err != nil {
			glog.Errorf("Failed to read checkpoint source: %v", err)
			continue
		}

		dst := PodFullNameToActiveCheckpointPath(id)
		if err := ioutil.WriteFile(dst, data, 0644); err != nil {
			glog.Errorf("Failed to write active checkpoint manifest: %v", err)
		}
	}
}

func isRunning(pod *v1.Pod) bool {
	// Determine if a pod is "running" by checking if each container status is in a "ready" state
	// TODO(aaron): Figure out best sets of data to inspect. PodConditions, PodPhase, containerStatus, containerState, etc.
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if !containerStatus.Ready {
			glog.Infof("Container %s in pod %s not ready. Will not checkpoint", containerStatus.Name, pod.Name)
			return false
		}
	}
	return true
}

func podListToParentPods(pl *v1.PodList) map[string]*v1.Pod {
	return podListToMap(pl, isValidParent)
}

func filterNone(p *v1.Pod) bool {
	return true
}

type filterFn func(*v1.Pod) bool

func podListToMap(pl *v1.PodList, filter filterFn) map[string]*v1.Pod {
	pods := make(map[string]*v1.Pod)
	for i := range pl.Items {
		if !filter(&pl.Items[i]) {
			continue
		}

		pod := &pl.Items[i]
		id := PodFullName(pod)

		if _, ok := pods[id]; ok { // TODO(aaron): likely not be necessary (shouldn't ever happen) - but sanity check
			glog.Warningf("Found multiple local parent pods with same id: %s", id)
		}

		// Pods from Kubelet API do not have TypeMeta populated - set it here either way.
		pods[id] = pod
		pods[id].TypeMeta = unversioned.TypeMeta{
			APIVersion: pl.APIVersion,
			Kind:       "Pod",
		}
	}
	return pods
}

// A valid checkpoint parent:
//    has the checkpoint=true annotation
//    is not a static pod itself
//    is not a checkpoint pod itself
func isValidParent(pod *v1.Pod) bool {
	if pod.Annotations == nil {
		return false
	}
	shouldCheckpoint := pod.Annotations[shouldCheckpointAnnotation] == shouldCheckpoint
	isStatic := pod.Annotations[podSourceAnnotation] == podSourceFile
	return shouldCheckpoint && !isStatic && !isCheckpoint(pod)
}

func isCheckpoint(pod *v1.Pod) bool {
	if pod.Annotations == nil {
		return false
	}
	_, ok := pod.Annotations[checkpointParentAnnotation]
	return ok
}

func copyPod(pod *v1.Pod) (*v1.Pod, error) {
	obj, err := api.Scheme.Copy(pod)
	if err != nil {
		return nil, err
	}
	return obj.(*v1.Pod), nil
}

func PodFullName(pod *v1.Pod) string {
	return pod.Namespace + "/" + pod.Name
}

func PodFullNameToInactiveCheckpointPath(id string) string {
	return filepath.Join(inactiveCheckpointPath, strings.Replace(id, "/", "-", -1)+".json")
}

func PodFullNameToActiveCheckpointPath(id string) string {
	return filepath.Join(activeCheckpointPath, strings.Replace(id, "/", "-", -1)+".json")
}

func secretPath(namespace, podName, secretName string) string {
	return filepath.Join(checkpointSecretPath, namespace, podName, secretName)
}

func PodFullNameToSecretPath(id string) string {
	namespace, podname := path.Split(id)
	return filepath.Join(checkpointSecretPath, namespace, podname)
}

func newClient() clientset.Interface {
	// This is run as a static pod, so we can't use InClusterConfig (no secrets/service account)
	// Use the same kubeconfig as the kubelet for auth and api-server location.
	kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		glog.Fatalf("Failed to load kubeconfig: %v", err)
	}
	return clientset.NewForConfigOrDie(kubeConfig)
}

func writeAndAtomicRename(path string, data []byte, perm os.FileMode) error {
	tmpfile := filepath.Join(filepath.Dir(path), "."+filepath.Base(path))
	if err := ioutil.WriteFile(tmpfile, data, perm); err != nil {
		return err
	}
	return os.Rename(tmpfile, path)
}
