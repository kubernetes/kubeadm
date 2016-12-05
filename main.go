package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_4"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	clientcmdapi "k8s.io/kubernetes/pkg/client/unversioned/clientcmd/api"
)

const (
	kubeletAPIPodsURL = "http://127.0.0.1:10255/pods"
	ignorePath        = "/srv/kubernetes/manifests"
	activePath        = "/etc/kubernetes/manifests"
	kubeconfigPath    = "/etc/kubernetes/kubeconfig"
	secretsPath       = "/etc/kubernetes/checkpoint-secrets"

	tempAPIServer = "temp-apiserver"
	kubeAPIServer = "kube-apiserver"
)

var podAPIServerMeta = unversioned.TypeMeta{
	APIVersion: "v1",
	Kind:       "Pod",
}

var (
	secureAPIAddr = fmt.Sprintf("https://%s:%s", os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT_HTTPS"))
)

func main() {
	glog.Info("begin pods checkpointing...")
	run(kubeAPIServer, tempAPIServer, api.NamespaceSystem)
}

func run(actualPodName, tempPodName, namespace string) {
	client := newAPIClient()
	for {
		var podList v1.PodList
		if err := json.Unmarshal(getPodsFromKubeletAPI(), &podList); err != nil {
			glog.Fatal(err)
		}
		switch {
		case bothRunning(podList, actualPodName, tempPodName, namespace):
			glog.Infof("both temp %v and actual %v pods running, removing temp pod", actualPodName, tempPodName)
			// Both the temp and actual pods are running.
			// Remove the temp manifest from the config dir so that the
			// kubelet will stop it.
			if err := os.Remove(activeManifest(tempPodName)); err != nil {
				glog.Error(err)
			}
		case isPodRunning(podList, client, actualPodName, namespace):
			glog.Infof("actual pod %v found, creating temp pod manifest", actualPodName)
			// The actual is running. Let's snapshot the pod,
			// clean it up a bit, and then save it to the ignore path for
			// later use.
			checkpointPod := createCheckpointPod(podList, actualPodName, namespace)
			convertSecretsToVolumeMounts(client, &checkpointPod)
			writeManifest(checkpointPod, tempPodName)
			glog.Infof("finished creating temp pod %v manifest at %s\n", tempPodName, checkpointManifest(tempPodName))

		default:
			glog.Info("no actual pod running, installing temp pod static manifest")
			b, err := ioutil.ReadFile(checkpointManifest(tempPodName))
			if err != nil {
				glog.Error(err)
			} else {
				if err := ioutil.WriteFile(activeManifest(tempPodName), b, 0644); err != nil {
					glog.Error(err)
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func stripNonessentialInfo(p *v1.Pod) {
	p.Spec.ServiceAccountName = ""
	p.Spec.DeprecatedServiceAccount = ""
	p.Status.Reset()
}

func getPodsFromKubeletAPI() []byte {
	var pods []byte
	res, err := http.Get(kubeletAPIPodsURL)
	if err != nil {
		glog.Error(err)
		return pods
	}
	pods, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		glog.Error(err)
	}
	return pods
}

func bothRunning(pods v1.PodList, an, tn, ns string) bool {
	var actualPodSeen, tempPodSeen bool
	for _, p := range pods.Items {
		actualPodSeen = actualPodSeen || isPod(p, an, ns)
		tempPodSeen = tempPodSeen || isPod(p, tn, ns)
		if actualPodSeen && tempPodSeen {
			return true
		}
	}
	return false
}

func isPodRunning(pods v1.PodList, client clientset.Interface, n, ns string) bool {
	for _, p := range pods.Items {
		if isPod(p, n, ns) {
			if n == kubeAPIServer {
				// Make sure it's actually running. Sometimes we get that
				// pod manifest back, but the server is not actually running.
				_, err := client.Discovery().ServerVersion()
				return err == nil
			}
			return true
		}
	}
	return false
}

func isPod(pod v1.Pod, n, ns string) bool {
	return strings.Contains(pod.Name, n) && pod.Namespace == ns
}

// cleanVolumes will sanitize the list of volumes and volume mounts
// to remove the default service account token.
func cleanVolumes(p *v1.Pod) {
	volumes := make([]v1.Volume, 0, len(p.Spec.Volumes))
	for _, v := range p.Spec.Volumes {
		if !strings.HasPrefix(v.Name, "default-token") {
			volumes = append(volumes, v)
		}
	}
	p.Spec.Volumes = volumes
	for i := range p.Spec.Containers {
		c := &p.Spec.Containers[i]
		volumeMounts := make([]v1.VolumeMount, 0, len(c.VolumeMounts))
		for _, vm := range c.VolumeMounts {
			if !strings.HasPrefix(vm.Name, "default-token") {
				volumeMounts = append(volumeMounts, vm)
			}
		}
		c.VolumeMounts = volumeMounts
	}
}

// writeManifest will write the manifest to the ignore path.
// It first writes the file to a temp file, and then atomically moves it into
// the actual ignore path and correct file name.
func writeManifest(manifest v1.Pod, name string) {
	m, err := json.Marshal(manifest)
	if err != nil {
		glog.Fatal(err)
	}
	writeAndAtomicCopy(m, checkpointManifest(name))
}

func createCheckpointPod(podList v1.PodList, n, ns string) v1.Pod {
	var checkpointPod v1.Pod
	for _, p := range podList.Items {
		if isPod(p, n, ns) {
			checkpointPod = p
			break
		}
	}
	// the pod we manifest we got from kubelet does not have TypeMeta.
	// Add it now.
	checkpointPod.TypeMeta = podAPIServerMeta
	cleanVolumes(&checkpointPod)
	stripNonessentialInfo(&checkpointPod)
	return checkpointPod
}

func newAPIClient() clientset.Interface {
	kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: secureAPIAddr}}).ClientConfig()
	if err != nil {
		glog.Fatal(err)
	}
	return clientset.NewForConfigOrDie(kubeConfig)
}

func convertSecretsToVolumeMounts(client clientset.Interface, pod *v1.Pod) {
	glog.Info("converting secrets to volume mounts")
	spec := pod.Spec
	for i := range spec.Volumes {
		v := &spec.Volumes[i]
		if v.Secret != nil {
			secretName := v.Secret.SecretName
			basePath := filepath.Join(secretsPath, pod.Name, v.Secret.SecretName)
			v.HostPath = &v1.HostPathVolumeSource{
				Path: basePath,
			}
			copySecretsToDisk(client, secretName, basePath)
			v.Secret = nil
		}
	}
}

func copySecretsToDisk(client clientset.Interface, secretName, basePath string) {
	glog.Info("copying secrets to disk")
	if err := os.MkdirAll(basePath, 0755); err != nil {
		glog.Fatal(err)
	}
	glog.Infof("created directory %s", basePath)
	s, err := client.Core().Secrets(api.NamespaceSystem).Get(secretName)
	if err != nil {
		glog.Fatal(err)
	}
	for name, value := range s.Data {
		path := filepath.Join(basePath, name)
		writeAndAtomicCopy(value, path)
	}
}

func writeAndAtomicCopy(data []byte, path string) {
	// First write a "temp" file.
	tmpfile := filepath.Join(filepath.Dir(path), "."+filepath.Base(path))
	if err := ioutil.WriteFile(tmpfile, data, 0644); err != nil {
		glog.Fatal(err)
	}
	// Finally, copy that file to the correct location.
	if err := os.Rename(tmpfile, path); err != nil {
		glog.Fatal(err)
	}
}

func activeManifest(name string) string {
	return filepath.Join(activePath, name+".json")
}

func checkpointManifest(name string) string {
	return filepath.Join(ignorePath, name+".json")
}
