module k8s.io/kubeadm/operator

go 1.12

require (
	github.com/go-logr/logr v0.1.0
	github.com/pkg/errors v0.8.1
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/klog v0.3.1
	k8s.io/utils v0.0.0-20190506122338-8fab8cb257d5
	sigs.k8s.io/cluster-api v0.0.0-20190820130448-9fd8e4cbea0f
	sigs.k8s.io/controller-runtime v0.2.2
)
