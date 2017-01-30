package main

type MasterConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// Data shared between phases. Used for defaulting
	// TODO: How should we deal with this?
	// Should they be both top-level and in-phase-level?
	Networking Networking `json:"networking"`
	CertificatesDir string `json:"certificatesDir"`

	// Phases
	Certificates     Certificates     `json:"certificates"`
	KubeConfig       KubeConfig       `json:"kubeConfig"`
	ControlPlane     ControlPlane     `json:"controlPlane"`
	Discovery        MasterDiscovery        `json:"discovery"`
	APIConfiguration APIConfiguration `json:"apiConfiguration"`
	Addons           Addons           `json:"addons"`
}

type Phase struct {
	Annotations map[string]string `json:"annotations"`
}

// TODO: This is not a good struct, we should probably get rid of it
type Networking struct {
	// This sets --service-cluster-ip-range on the api server
	// Default: 10.96.0.1/12
	ServiceSubnet string `json:"serviceSubnet"`
	// Default: none => kube-proxy doesn't know which traffic is external and which is internal
	// We should maybe require this value
	PodSubnet string `json:"podSubnet"`
	// Default: cluster.local
	DNSDomain string `json:"dnsDomain"`
	// We should maybe require this value
	CNIFilePath string `json:"cniFilePath"`
}

// Phases with their subtypes
type Certificates struct {
	Phase Phase `json:"phase"`

	// In the future, we may provide more options for generating certs
	// For instance, one can use Vault for storing the certs
	SelfSign *SelfSignCertificates `json:"selfSign"`
}

type SelfSignCertificates struct {
	// Needs these fields in Master, Networking and Paths
	// Inputs
	ServiceSubnet string `json:"serviceSubnet"`
	DNSDomain string `json:"dnsDomain"`

	// All IP addresses and DNS names these certs should be signed for
	// Defaults to the default networking interface's IP address and the hostname of the master node
	AltNames []string `json:"altNames"`
	// For example, let the user choose key type
	// Can be RSA, ECDSA (or Ed25519 in the future)
	// Default: RSA
	// TODO: Maybe we should omit this option and let the user self-sign in case of ECDSA or Ed25519?
	PrivateKeyType string `json:"privateKeyType"`

	// Outputs
	CertificatesDir string `json:"certificatesDir"`
}

type KubeConfig struct {
	Phase Phase `json:"phase"`

	// TODO: Find a better name/purpose
	MasterDefault *MasterDefaultKubeConfig `json:"masterDefault"`
}

// TODO: Decide whether this phase should generate both for kubelet and admin.
// This phase is probably useful generally for creating kubeconfigs from certificates
type MasterDefaultKubeConfig struct {
	// Inputs
	// Path where the certs are located
	CertificatesDir string `json:"certificatesDir"`
	// This could be a []string in the API, but initially only support a string before KubeConfig itself supports multiple endpoints.
	MasterEndpoints []string `json:"masterEndpoints"`

	// Outputs
	// The path to where the admin kubeconfig file should be written
	// TODO: Do we want to expose these two values or should they just be hardcoded as /etc/kubernetes/admin.conf and /etc/kubernetes/kubelet.conf
	AdminConfigPath string `json:"adminConfigPath"`
	// We should be able to generate this KubeConfig file in the same manner as we do on nodes, so the master kubelets don't
	// have full access to the apiserver while the node kubelet would have limited access, which is a thing we should do later.
	KubeletConfigPath string `json:"kubeletConfigPath"`
}

type ControlPlane struct {
	Phase Phase `json:"phase"`

	// Needs these fields in Networking
	CertificatesDir string `json:"certificatesDir"`

	// Networking kind of stuff
	ServiceSubnet string `json:"serviceSubnet"`
	DNSDomain     string `json:"dnsDomain"`
	// This has to be solved somehow. kube-proxy needs the podsubnet and if allocateNodeCIDRs is true, controller-manager also needs it
	PodSubnet     string `json:"podSubnet"`
	// Whether controller-manager should allocate cidrs to nodes
	// TODO: We might want to always set this
	AllocateNodeCIDRs bool `json:"allocateNodeCIDRs"`

	// Defaults the latest stable version
	Version           string `json:"version"`
	// Defaults to gcr.io/google_containers
	ImageRepository   string `json:"imageRepository"`
	// This makes it possible to override the control plane images to using
	// one hyperkube image only
	UseHyperkubeImage string `json:"useHyperkubeImage"`
	// Defaults to 6443 in order to not conflict with normal HTTP/HTTPS traffic if any
	// Also, the user might want to deploy an ingress controller on the master in bare-metal solutions, and therefore we'd not like to default to 443
	APIServerPort uint32 `json:"apiServerPort"`
	// TODO: Should we allow more than one here?
	// Currently, the apiserver itself doesn't allow more than one, but we might want to be future-proof
	// I guess we could leave this as a string while in beta
	APIServerBindAddress string `json:"apiServerBindAddress"`

	// Specifies which authorization mode the apiserver should use
	AuthorizationMode string `json:"authorizationMode"`

	// Right now, it can take values like "aws" or "gce", but in the future, that may change to take a path to a file with the cloudprovider controller manifest: https://github.com/kubernetes/community/pull/128
	CloudProvider string `json:"cloudProvider"`

	// Specifies how to deploy or connect to etcd
	Etcd Etcd `json:"etcd"`

	// TODO: We want to use ComponentConfig here eventually
	ExtraArguments       ComponentExtraList `json:"extraArguments"`
	ExtraHostPathVolumes ComponentExtraList `json:"extraHostPathVolumes"`

	// Only one of StaticPod and SelfHosted can be defined
	StaticPod  *StaticPodControlPlane  `json:"staticPod"`
	SelfHosted *SelfHostedControlPlane `json:"selfHosted"`
}

type StaticPodControlPlane struct {
	// TODO: What options do we need here?
	Dummy string
}

type SelfHostedControlPlane struct {
	ControllerManagerAndSchedulerReplicas uint8
}

type ComponentExtraList struct {
	APIServer         []string `json:"apiServer"`
	ControllerManager []string `json:"controllerManager"`
	Scheduler         []string `json:"scheduler"`
}

type Etcd struct {
	// Only one of External and Local can be defined
	External *ExternalEtcd `json:"external"`
	Local    *LocalEtcd    `json:"local"`
}

type ExternalEtcd struct {
	Endpoints []string `json:"endpoints"`
	CAFile    string   `json:"caFile"`
	CertFile  string   `json:"certFile"`
	KeyFile   string   `json:"keyFile"`
}

type LocalEtcd struct {
	DataDir string `json:"dataDir"`
	Image   string `json:"image"`
}

type MasterDiscovery struct {
	Phase Phase `json:"phase"`

	// Only one of File and Token can be defined
	// File outputs a kubeconfig file that can directly be used as an input to the HTTPS or File-based node discovery
	File  *FileDiscovery  `json:"file"`
	// Token enables support for the token-based discovery
	Token *TokenDiscovery `json:"token"`
}

type HTTPSDiscovery struct {
	URL string `json:"url"`
}

type FileDiscovery struct {
	Path string `json:"path"`
}

type TokenDiscovery struct {
	ID        string   `json:"id"`
	Secret    string   `json:"secret"`
	Addresses []string `json:"addresses"`
}

type APIConfiguration struct {
	Phase Phase `json:"phase"`

	// Defaults to /etc/kubernetes/admin.conf
	KubeConfigFile string `json:"kubeConfigFile"`

	// If specified, kubeadm will exec out to an other binary
	// Is this a sane thing to do security-wise?
	ExecHook v1.ExecAction

	// Custom configmaps the user would like to inject into the kube-system namespace
	// Do we need this?
	ConfigMaps map[string][]byte `json:"kubeSystemConfigMaps"`

	// A yaml or json componentconfig that sets the base layer for kubelet configuration across the cluster
	KubeletBaseConfiguration []byte `json:"kubeletBaseConfiguration"`

	// If the authorization mode is RBAC; kubeadm will set up some default rules
	AuthorizationMode string `json:"authorizationMode"`

	// We should maybe require this value
	CNIFilePath string `json:"cniFilePath"`
	// Default: none => kube-proxy doesn't know which traffic is external and which is internal
	// We should maybe require this value
	PodSubnet string `json:"podSubnet"`
}

type Addons struct {
	Phase Phase `json:"phase"`

	// Defaults to /etc/kubernetes/admin.conf
	KubeConfigFile string `json:"kubeConfigFile"`

	// TODO: This is probably not needed
	ServiceSubnet string `json:"serviceSubnet"`

	DNSDomain     string `json:"dnsDomain"`

	ImageRepository string `json:"imageRepository"`

	ApplyManifests []string `json:"applyManifests"`
}

type NodeConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	Discovery NodeDiscovery `json:"discovery"`
}

type NodeDiscovery struct {
	// Only one of HTTPS, File and Token can be defined
	HTTPS *HTTPSDiscovery `json:"https"`
	File  *FileDiscovery  `json:"file"`
	Token *TokenDiscovery `json:"token"`
}
