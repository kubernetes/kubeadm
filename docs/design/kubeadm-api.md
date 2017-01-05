## Kubeadm Phases and API

### Refactoring kubeadm
We all want kubeadm modular and componentized; here’s the proposal for that!
I’m not writing down all reasons for doing this; they are already available in the blog post: https://docs.google.com/document/d/1bpEGhdg8t-2Ae-IL_ezZIPklMHpZLWwnoAPMgLwT3oU

### Phases
We want to identify the phases every single Kubernetes deployment goes through in order to get the cluster up and running.

In this process, I've been thinking a lot about security and tried to put that first. Also, I've read the 
kops, kargo and kube-up code in detail to see how they are doing these things. We don't want to make an API that's incompatible with those tools.

#### Constants

Somewhere we have to draw the line what should be configurable, what shouldn't and what should be hard-coded in the binary.

We've decided to make the Kubernetes directory `/etc/kubernetes` a constant in the application, since it is clearly the given path in a lot of places,
and the most intuitive location. Having that path configurable would for sure confuse readers of a on-top-of-kubeadm-implemented deployment solution.

#### Phase API

The user should be able to pass a config file containing a `MasterConfiguration` API object, and kubeadm will run the 


So far (for kubeadm) I've identified these:

#### Certificates

First, certificates should be created in a directory.
There should be:
 - a CA certificate (`ca.crt`) with its private key (`ca.key`)
 - an API Server certificate (`apiserver.crt`) using `ca.crt` as the CA with its private key (`apiserver.key`)
   - the API Server private key will also function as the private key for generating ServiceAccount secrets

There should be tests to make sure that RSA, ECDSA and maybe Ed25519 private keys can be generated with kubeadm.

If the `ca.crt` and the `ca.key` both exist in the directory that's given, which defaults to `/etc/kubernetes/pki`,
the CA generation will be skipped, and those CA credentials will be used. If all four files _exist, are readable and have not expired_
kubeadm will print out that it is using the certificates that already exist, otherwise it will return an error if for instance 
`ca.crt` exists but have expired.

Specifying an own CA might be beneficial to orgs that require a specific CA but lets the apiserver keys/certs be generated on the go by kubeadm,
so that's a hard requirement.

TODO: How should the apiserver <-> kubelet communication be secured properly?

**Inputs**
 - `ExtraDomainsAndAddresses` is needed for knowing which IPs and DNS names the certs should be signed for
 - `DNSDomain` is needed for knowing which DNS name the internal kubernetes service has
 - `ServiceSubnet` is needed for knowing which IP the internal kubernetes service is going to point to
 - `CertificatesDir` is required for knowing where all certificates should be stored

**Outputs**
 - Files in directory `CertificatesDir`:
   - `ca.crt`
   - `ca.key`
   - `apiserver.crt`
   - `apiserver.key`

#### Generate KubeConfig files for the admin and the kubelet on the master

The second phase takes the certificates generated in phase 1 and the public master endpoint as input and produces two kubeconfig files,
`admin.conf` and `kubelet.conf` in `/etc/kubernetes/`.

This phase 

There also is the question whether this phase should generate both the kubelet and the admin config. Or should those be two different phases/steps?
How can we make the kubelet have just a little permissions but still use it for bootstrapping the api server? Can start the master ("bootstrap") kubelet with full access, 
then do the CSR dance and swap the kubeconfig later on the fly?

**Inputs**
 - `CertificatesDir` is required for knowing where the certificates are stored
 - `MasterEndpoint` is needed for knowing where to find the API Server (or servers if using a DNS endpoint)

**Outputs**
 - Files in directory `KubernetesDir`:
   - `admin.conf`
   - `kubelet.conf`

#### Bring up the control plane

This is probably the most complex task with lots of configurability options.
Ideally, we'd have the ComponentConfig API ready to use, but unfortunately it is not.

First off, we have two modes currently for bootstrapping etcd: `local` and `external`.
They have straightforward API types:

```go
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
```

**Inputs**
 - `CertificatesDir` is required for knowing where all certificates are stored
 - `MasterEndpoint` is needed for knowing where to find the API Server (servers if using a DNS endpoint)
 - `KubernetesDir` is needed for knowing where to put the kubeconfig files. Should be constant or not?

#### Configure the API Server

TODO
Add users, configmaps, taints etc.

#### Setup Discovery

TODO

#### Bring up addons

TODO

In particular, kube-dns and kube-proxy

Use `--kubernetes-service-node-port` here and let the kube-proxy use the built-in SA certs?


### Proposed API types with respect to the information above

