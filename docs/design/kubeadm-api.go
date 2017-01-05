package main

type MasterConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// Data shared between phases. Used for defaulting
	Master     Master     `json:"master"`
	Networking Networking `json:"networking"`
	CertificatesDir string `json:"certificatesDir"`

	// Phases
	Certificates     Certificates     `json:"certificates"`
	KubeConfig       KubeConfig       `json:"kubeConfig"`
	ControlPlane     ControlPlane     `json:"controlPlane"`
	Discovery        Discovery        `json:"discovery"`
	APIConfiguration APIConfiguration `json:"apiConfiguration"`
	Addons           Addons           `json:"addons"`
}

type Phase struct {
	Annotations map[string]string `json:"annotations"`
}

// Shared
// TODO: This field should be HA friendly
type Master struct {
	// Only the first address here will be passed to the api-server. The rest will be used for signing CA certs.
	// This is not great with HA, because can't
	AdvertiseAddresses []string `json:"advertiseAddresses"`
	// Used for signing the certs
	ExternalDNSNames []string `json:"externalDNSNames"`
	// For the controlplane phase
	Port int32 `json:"port"`
}

type Networking struct {
	// This sets --service-cluster-ip-range on the api server
	// Default: 10.96.0.1/12
	ServiceSubnet string `json:"serviceSubnet"`
	// Default: none => disable this functionality
	// TODO: Test if this always can be set
	PodSubnet string `json:"podSubnet"`
	// Default: cluster.local
	DNSDomain string `json:"dnsDomain"`
	// Possible: cni|kubenet. Default: cni
	NetworkPlugin string `json:"networkPlugin"`
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
	// ServiceSubnet string `json:"serviceSubnet"`
	// DNSDomain string `json:"dnsDomain"`
	// Defaults to .Networking
	Networking Networking `json:"networking"`

	// All IP addresses and DNS names these certs should be signed for
	// Defaults to an union of .Master.AdvertiseAddresses and .Master.ExternalDNSNames and the hostname of the master node
	// This could also be named AltNames if we want
	ExtraDomainsAndAddresses []string `json:"extraDomainsAndAddresses"`
	// For example, let the user choose key type
	// Can be RSA, ECDSA or Ed25519
	// Default: RSA
	PrivateKeyType string `json:"privateKeyType"`

	// Outputs
	CertificatesDir string `json:"certificatesDir"`
}

type KubeConfig struct {
	Phase Phase `json:"phase"`

	//
	MasterDefault *MasterDefaultKubeConfig `json:"masterDefault"`
}

// TODO: Decide whether this phase should generate both for kubelet and admin.
// This phase is probably useful generally for creating kubeconfigs from certificates
type MasterDefaultKubeConfig struct {
	// Inputs
	// Path where the certs are located
	CertificatesDir string `json:"certificatesDir"`
	// This could be a []string in the API, but initially only support a string before KubeConfig itself supports multiple endpoints.
	MasterEndpoint []string `json:"masterEndpoint"`

	// Outputs
	// The path to where the admin kubeconfig file should be written
	// Hmm: Do we want to expose these two values or should they just be hardcoded as /etc/kubernetes/admin.conf and /etc/kubernetes/kubelet.conf
	AdminConfigPath string `json:"adminConfigPath"`
	// We should be able to generate this KubeConfig file in the same manner as we do on nodes, so the master kubelets don't
	// have full access to the apiserver while the node kubelet would have limited access, which is a thing we should do later.
	KubeletConfigPath string `json:"kubeletConfigPath"`
}

type ControlPlane struct {
	Phase Phase `json:"phase"`

	// Needs these fields in Master, Networking and Paths
	// AdvertiseAddresses []string `json:"advertiseAddresses"`
	// ExternalDNSNames []string `json:"externalDNSNames"`
	// Port int32 `json:"port"`
	// CertificatesDir string `json:"certificatesDir"`
	// ServiceSubnet string `json:"serviceSubnet"`
	// DNSDomain     string `json:"dnsDomain"`
	// PodSubnet     string `json:"podSubnet"`

	Version           string `json:"version"`
	ImageRepository   string `json:"imageRepository"`
	UseHyperkubeImage string `json:"useHyperkubeImage"`

	// Deprecated and will be removed soon
	CloudProvider string `json:"cloudProvider"`

	Etcd Etcd `json:"etcd"`

	// TODO: We want to use ComponentConfig here eventually
	ExtraArguments       ComponentExtraList `json:"extraArguments"`
	ExtraHostPathVolumes ComponentExtraList `json:"extraHostPathVolumes"`

	// Only one of StaticPod and SelfHosted can be defined
	StaticPod  *StaticPodControlPlane  `json:"staticPod"`
	SelfHosted *SelfHostedControlPlane `json:"selfHosted"`
}

type StaticPodControlPlane struct {
	Dummy string
}

type SelfHostedControlPlane struct {
	Dummy string
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

type Discovery struct {
	Phase Phase `json:"phase"`

	// Only one of HTTPS, File and Token can be defined
	HTTPS *HTTPSDiscovery `json:"https"`
	File  *FileDiscovery  `json:"file"`
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

	KubeConfigFile string `json:"kubeConfigFile"`

	KubeSystemConfigMaps map[string]string `json:"kubeSystemConfigMaps"`
}

type Addons struct {
	Phase Phase `json:"phase"`

	KubeConfigFile string `json:"kubeConfigFile"`

	ImageRepository string `json:"imageRepository"`

	ApplyManifests []string `json:"applyManifests"`
}

type NodeConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	Discovery Discovery `json:"discovery"`
}
