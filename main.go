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
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_2"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	clientcmdapi "k8s.io/kubernetes/pkg/client/unversioned/clientcmd/api"
)

const (
	kubeletAPIPodsURL = "http://localhost:10255/pods"
	ignorePath        = "/srv/kubernetes/manifests"
	activePath        = "/etc/kubernetes/manifests"
	manifestFilename  = "apiserver.json"
	kubeconfigPath    = "/etc/kubernetes/kubeconfig"
	secretsPath       = "/etc/kubernetes/checkpoint-secrets"
)

var (
	tempAPIServer      = []byte("temp-apiserver")
	kubeAPIServer      = []byte("kube-apiserver")
	activeManifest     = filepath.Join(activePath, manifestFilename)
	checkpointManifest = filepath.Join(ignorePath, manifestFilename)
	secureAPIAddr      = fmt.Sprintf("https://%s:%s", os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT_HTTPS"))
)

var tempAPIServerManifest = v1.Pod{
	TypeMeta: unversioned.TypeMeta{
		APIVersion: "v1",
		Kind:       "Pod",
	},
	ObjectMeta: v1.ObjectMeta{
		Name:      "temp-apiserver",
		Namespace: "kube-system",
	},
}

func main() {
	glog.Info("begin apiserver checkpointing...")
	run()
}

func run() {
	client := newAPIClient()
	for {
		var podList v1.PodList
		if err := json.Unmarshal(getPodsFromKubeletAPI(), &podList); err != nil {
			glog.Fatal(err)
		}
		switch {
		case bothAPIServersRunning(podList):
			glog.Info("both temp and kube apiserver running, removing temp apiserver")
			// Both the self-hosted API Server and the temp API Server are running.
			// Remove the temp API Server manifest from the config dir so that the
			// kubelet will stop it.
			if err := os.Remove(activeManifest); err != nil {
				glog.Error(err)
			}
		case kubeSystemAPIServerRunning(podList):
			glog.Info("kube-apiserver found, creating temp-apiserver manifest")
			// The self-hosted API Server is running. Let's snapshot the pod,
			// clean it up a bit, and then save it to the ignore path for
			// later use.
			tempAPIServerManifest.Spec = parseAPIPodSpec(podList)
			convertSecretsToVolumeMounts(client, &tempAPIServerManifest)
			writeManifest(tempAPIServerManifest)
			glog.Infof("finished creating temp-apiserver manifest at %s\n", checkpointManifest)

		default:
			glog.Info("no apiserver running, installing temp apiserver static manifest")
			b, err := ioutil.ReadFile(checkpointManifest)
			if err != nil {
				glog.Error(err)
			} else {
				if err := ioutil.WriteFile(activeManifest, b, 0644); err != nil {
					glog.Error(err)
				}
			}
		}
		time.Sleep(60 * time.Second)
	}
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

func bothAPIServersRunning(pods v1.PodList) bool {
	var kubeAPISeen, tempAPISeen bool
	for _, p := range pods.Items {
		kubeAPISeen = kubeAPISeen || isKubeAPI(p)
		tempAPISeen = tempAPISeen || isTempAPI(p)
		if kubeAPISeen && tempAPISeen {
			return true
		}
	}
	return false
}

func kubeSystemAPIServerRunning(pods v1.PodList) bool {
	for _, p := range pods.Items {
		if isKubeAPI(p) {
			return true
		}
	}
	return false
}

func isKubeAPI(pod v1.Pod) bool {
	return strings.Contains(pod.Name, "kube-apiserver") && pod.Namespace == api.NamespaceSystem
}

func isTempAPI(pod v1.Pod) bool {
	return strings.Contains(pod.Name, "temp-apiserver") && pod.Namespace == api.NamespaceSystem
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

// We need to ensure that the temp apiserver is not binding
// on the same insecure port as our self-hosted apiserver, otherwise
// it will exit immediately instead of waiting to bind.
func modifyInsecurePort(p *v1.Pod) {
	for i := range p.Spec.Containers {
		c := &p.Spec.Containers[i]
		cmds := c.Command
		for i, cmd := range cmds {
			if strings.Contains(cmd, "insecure-port") {
				cmds[i] = strings.Replace(cmd, "8080", "8081", 1)
				break
			}
		}
	}
}

// writeManifest will write the manifest to the ignore path.
// It first writes the file to a temp file, and then atomically moves it into
// the actual ignore path and correct file name.
func writeManifest(manifest v1.Pod) {
	m, err := json.Marshal(manifest)
	if err != nil {
		glog.Fatal(err)
	}
	writeAndAtomicCopy(m, checkpointManifest)
}

func parseAPIPodSpec(podList v1.PodList) v1.PodSpec {
	var apiPod v1.Pod
	for _, p := range podList.Items {
		if isKubeAPI(p) {
			apiPod = p
			break
		}
	}
	cleanVolumes(&apiPod)
	modifyInsecurePort(&apiPod)
	return apiPod.Spec
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
			break
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
