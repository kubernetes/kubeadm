module KubeadmOptions

    # How the controlplane will be deployed?
    class Controlplane
        StaticPods  = 'staticPods'    
        SelfHosting = 'selfHosting'

        All         = [StaticPods, SelfHosting]

        def self.validate(string)
            unless [StaticPods, SelfHosting].include?(string)
                raise "invalid controlplane type '#{string}'. Valid types are [#{All.join(', ')}]." 
            end
        end
    end

    # Which type of high availability for the controlplane are you planning?
    # NB. this is implicitily derived from the machineSets:
    # SingleMaster if only one machine with role Master exists, MultiMaster if more than one machine with role Master exists
    class ControlplaneAvailability
        SingleMaster    = 'singleMaster'        
        MultiMaster     = 'multiMaster'

        All             = [SingleMaster, MultiMaster]
    end

    # How etcd will be deployed?
    # NB. this is implicitily derived from the machineSets:
    # External if at least one machine with role etcd exists, otherwise local
    class EtcdMode
        Local       = 'local'        
        External    = 'external'

        All         = [Local, External]
    end

    # Which type of certificate authority are you going to use?
    class CertificateAuthority
        Local       = 'local'
        External    = 'external' 

        All         = [Local, External]

        def self.validate(string)
            unless All.include?(string)
                raise "invalid CertificateAuthority type '#{string}'. Valid types are [#{All.join(', ')}]." 
            end
        end
    end

    # Where you PKI will be stored?
    class PKILocation
        Filesystem = 'filesystem'   
        Secrets = 'secrets'

        All    = [Filesystem, Secrets]

        def self.validate(string)
            unless All.include?(string)
                raise "invalid PKILocation '#{string}'. Valid locations are [#{All.join(', ')}]." 
            end
        end
    end

    DNSDomain = "cluster.local"

    # Which type of DNS you will use?
    class DNSAddon
        KubeDNS     = 'kubeDNS'      
        CoreDNS     = 'coreDNS' 

        All         = [KubeDNS, CoreDNS]

        def self.validate(string)
            unless All.include?(string)
                raise "invalid DNS add-on '#{string}'. Valid add-ons are [#{All.join(', ')}]." 
            end
        end
    end

    # Which type of pod network add-on are you using?
    class NetworkAddon
        Weavenet    = 'weavenet' 
        Flannel     = 'flannel'      
        Calico      = 'calico' 

        All         = [Weavenet, Flannel, Calico]

        def self.validate(string)
            unless All.include?(string)
                raise "invalid network add-on '#{string}'. Valid add-ons are [#{All.join(', ')}]." 
            end
        end

        def self.defaultSubnetsFor(addon)
            case addon
            when Weavenet
                return [nil], [nil]
            when Flannel
                return [nil], ["10.244.0.0/16"]
            when Calico
                return [nil], ["192.168.0.0/16"]
            end
        end
    end

    # Which type of kubelet config are you using?
    class KubeletConfig
        SystemdDropIn           = 'systemdDropIn'      
        DynamicKubeletConfig    = 'dynamicKubeletConfig' # technically this is SystemdDropIn + DynamicKubeletConfig

        All                     = [SystemdDropIn, DynamicKubeletConfig]

        def self.validate(string)
            unless All.include?(string)
                raise "invalid Kubelet config type '#{string}'. Valid types are [#{All.join(', ')}]." 
            end
        end
    end

    # TODO: 
    # Which type of network are you using (Ipv4, Ipv6)                                    --> test supports ipv4 only
    # Which type of environment are you simulating (with or without internet connection)  --> test supports with internet connection only
    # How Kube-proxy will be configured (Iptables, Ipvs)                                  --> test supports Iptables only
    # Auditing                                                                            --> test with auditing on
end