require 'yaml'

require_relative 'kubeadm-options.rb'
require_relative 'kubernetes-utils.rb'
require_relative 'vagrant-utils.rb'
require_relative 'ansible-utils.rb'
require_relative 'utils.rb'

require_relative 'test-guide.rb'

class VagrantMachine
    attr_accessor :name
    attr_accessor :hostname
    attr_accessor :box
    attr_accessor :ip
    attr_accessor :cpus
    attr_accessor :memory
    attr_accessor :roles
end

class VagrantMachineSet
    attr_accessor :name
    attr_accessor :replicas
    attr_accessor :box
    attr_accessor :cpus
    attr_accessor :memory
    attr_accessor :roles
end

class VagrantCluster
    
    attr_reader :name
    attr_reader :version
    attr_reader :controlplane
    attr_reader :controlplaneAvailability
    attr_reader :etcdMode
    attr_reader :certificateAuthority
    attr_reader :pkiLocation
    attr_reader :dnsAddon
    attr_reader :dnsDomain
    attr_reader :networkAddon
    attr_reader :serviceSubnet
    attr_reader :podSubnet
    attr_reader :kubeletConfig
    
    attr_reader :machines
    attr_reader :masters
    attr_reader :nodes
    attr_reader :etcds

    attr_reader :ansible_installEtcd
    attr_reader :ansible_setupExternalCA
    attr_reader :ansible_setupControlplaneVip
    attr_reader :ansible_kubeadmInit
    attr_reader :ansible_applyNetwork
    attr_reader :ansible_kubeadmJoinMaster
    attr_reader :ansible_kubeadmJoin

    attr_reader :ansible_groups
    attr_reader :ansible_extra_vars
    attr_reader :ansible_tags

    def self.fromClusterAPI (folder)
        c = VagrantCluster.new
        c.parseFolder folder
        c.compose
        return c
    end

    #$ ruby -r "./hack/lib/cluster-api.rb" -e "VagrantCluster.Test" up
    def self.Test
        c = fromClusterAPI('/Users/fabrizio/Documents/Workspace/Vagrant/kubernetes/dev/spec/*.yaml')
            .injectKubeadmFromBazelBuild(prefix: nil)
            .ansibleActions(
                installEtcd: true,              # only if machine with Etcd role are defined (external etcd)
                setupExternalCA: true,          # only if external certificate is requested
                setupControlplaneVip: true,     # only if there are 2 or more machines with Master role (multi master scenario)
                kubeadmInit: false, 
                applyNetwork: false,
                kubeadmJoinMaster: false, 
                kubeadmJoin: false
            )

        TestGuide.print(c)
    end

    def initialize
        @machineSets            = []
        @machines               = []
        @silent = VagrantUtils.silent
    end

    def parseFolder(folder)
        Utils.putsTitle "Reading cluster API spec from #{folder}" unless @silent
        Dir[folder].each do |file_name|
            next if File.directory? file_name 

            puts  "    - #{file_name}" unless @silent
            data = YAML.load_file(file_name)
  
            case data["kind"]
            when "Cluster"
                parseClusterData data
            when "MachineSet"
                parseMachineSetData data
            end
        end
    end

    def parseClusterData(data)
        @name                       = get data, "metadata", "name", validator: KuberentesUtils::DNS1123Label 
        @extra_vars                 = get data, "spec", "providerConfig", "value"

        @version                    = get data, "spec", "providerConfig", "value", "kubernetes", "version" #TODO: validate versione
        @controlplane               = get data, "spec", "providerConfig", "value", "kubernetes", "controlplane", default: KubeadmOptions::Controlplane::StaticPods, validator: KubeadmOptions::Controlplane
        # controlplaneAvailability will be implicitily derived from machineSets during compose
        # etcdMode will be implicitily derived from machineSets during compose
        @certificateAuthority       = get data, "spec", "providerConfig", "value", "kubernetes", "certificateAuthority", default: KubeadmOptions::CertificateAuthority::Local, validator: KubeadmOptions::CertificateAuthority
        @pkiLocation                = get data, "spec", "providerConfig", "value", "kubernetes", "pkiLocation", default: KubeadmOptions::PKILocation::Filesystem, validator: KubeadmOptions::PKILocation
        @dnsAddon                   = get data, "spec", "providerConfig", "value", "kubernetes", "dnsAddon", default: KubeadmOptions::DNSAddon::KubeDNS, validator: KubeadmOptions::DNSAddon
        @dnsDomain                  = get data, "spec", "clusterNetwork", "serviceDomain", default: KubeadmOptions::DNSDomain , validator: KuberentesUtils::DNS1123Subdomain
        @networkAddon               = get data, "spec", "providerConfig", "value", "kubernetes", "networkAddon", default: KubeadmOptions::NetworkAddon::Weavenet, validator: KubeadmOptions::NetworkAddon
        serviceSubnet, podSubnet    = KubeadmOptions::NetworkAddon.defaultSubnetsFor @networkAddon
        @serviceSubnet              = get data, "spec", "clusterNetwork", "services", "cidrblocks", default: serviceSubnet, validator: Utils::CIDRList
        @podSubnet                  = get data, "spec", "clusterNetwork", "pods", "cidrblocks", default: podSubnet, validator: Utils::CIDRList
        @kubeletConfig              = get data, "spec", "providerConfig", "value", "kubernetes", "kubeletConfig", default: KubeadmOptions::KubeletConfig::SystemdDropIn, validator: KubeadmOptions::KubeletConfig
        
        if @pkiLocation == KubeadmOptions::PKILocation::Secrets and @controlplane != KubeadmOptions::Controlplane::SelfHosted then raise "Invalid cluster definition. PKILocation can be secrets only if controlplane is self hosted" end
    end

    def parseMachineSetData(data)
        s = VagrantMachineSet.new 
        s.name        = get data, "metadata", "name", validator: KuberentesUtils::DNS1123Label
        s.replicas    = get data, "spec", "replicas", validator: Utils::Integer
        s.box         = get data, "spec", "template", "spec", "providerConfig", "value", "box"
        s.cpus        = get data, "spec", "template", "spec", "providerConfig", "value", "cpus", validator: Utils::Integer
        s.memory      = get data, "spec", "template", "spec", "providerConfig", "value", "memory", validator: Utils::Integer
        s.roles       = get data, "spec", "template", "spec", "roles", validator: Roles
        @machineSets << s
    end

    def compose
        @masters = []
        @etcds = []
        @nodes = []

        @machineSets.each do |s|
            for i in 1..s.replicas
                m = VagrantMachine.new
                m.name              = if s.replicas == 1 then "#{@name}-#{s.name}" else "#{@name}-#{s.name}#{i}" end  
                m.hostname          = "#{@name}-#{s.name}.local"
                m.box               = s.box
                m.ip                = "10.10.10.1#{i}"
                m.cpus              = s.cpus
                m.memory            = s.memory
                m.roles             = s.roles
                machines << m

                @masters << m if m.roles.include?(Roles::Master)
                @nodes   << m if m.roles.include?(Roles::Node)
                @etcds   << m if m.roles.include?(Roles::Etcd)
            end
        end

        if @masters.length == 0 then raise "Invalid cluster definition. At least one Master machine is required" end
        
        @etcdMode = KubeadmOptions::EtcdMode::Local
        if @etcds.length > 0 then @etcdMode = KubeadmOptions::EtcdMode::External end

        @controlplaneAvailability = KubeadmOptions::ControlplaneAvailability::SingleMaster 
        if @masters.length > 1 then 
            @controlplaneAvailability = KubeadmOptions::ControlplaneAvailability::MultiMaster 
            if @etcdMode != KubeadmOptions::EtcdMode::External then raise "Invalid cluster definition. Multi masters requires external etcd." end
            if @pkiLocation == KubeadmOptions::PKILocation::Secrets then raise "Invalid cluster definition. Multi masters does not support certificates in secrets yet." end
        end
        
        Utils.putsTitle "Machines defined in cluster API spec" unless @silent
        @machines.each do |m|
            puts "    - #{m.name}, roles: #{m.roles.join(",")} ]" unless @silent
        end
    end

    def get(data, *keys, default: nil, validator: nil)
        d = data
        keys.each do |k| 
            if d.has_key?(k)
                d = d[k]
            else
                if default!=nil 
                    return default
                else
                    raise "Error parsing #{keys} (#{k} does't exists)"
                end
            end
        end
        if validator!=nil 
            validator.validate d
        end
        return d
    end

    def injectKubeadmFromBazelBuild(prefix: nil)
        @ansible_extra_vars['kubeadm_binary'] = VagrantUtils::InjectKubeadmFromBuildOutput(build_output_path = "bazel-bin/cmd/kubeadm/linux_amd64_pure_stripped", prefix = prefix)
        return self
    end

    def injectKubeadmFromDockerBuild(prefix: nil)
        @ansible_extra_vars['kubeadm_binary'] = VagrantUtils::InjectKubeadmFromBuildOutput(build_output_path = "_output/dockerized/bin/linux/amd64/", prefix = prefix)
        return self
    end

    def injectKubeadmFromLocalBuild(prefix: nil)
        @ansible_extra_vars['kubeadm_binary'] = VagrantUtils::InjectKubeadmFromBuildOutput(build_output_path = "_output/local/bin/linux/amd64/", prefix = prefix)
        return self
    end

    def ansibleActions(installEtcd: true, setupExternalCA: true, setupControlplaneVip: true, kubeadmInit: false, applyNetwork: false, kubeadmJoinMaster: false, kubeadmJoin: false)    
        @ansible_installEtcd            = installEtcd
        @ansible_setupExternalCA        = setupExternalCA
        @ansible_setupControlplaneVip   = setupControlplaneVip
        @ansible_kubeadmInit            = kubeadmInit
        @ansible_applyNetwork           = applyNetwork
        @ansible_kubeadmJoinMaster      = kubeadmJoinMaster
        @ansible_kubeadmJoin            = kubeadmJoin

        @ansible_groups = AnsibleUtils::Inventory.for @machines, Roles::All
        @ansible_extra_vars = @extra_vars
        @ansible_extra_vars['kubernetes']['dnsDomain']     = @dnsDomain            unless @dnsDomain == KubeadmOptions::DNSDomain
        @ansible_extra_vars['kubernetes']['serviceSubnet'] = @serviceSubnet.first  unless @serviceSubnet.first == nil
        @ansible_extra_vars['kubernetes']['podSubnet']     = @podSubnet.first      unless @podSubnet.first == nil
        @ansible_tags   = AnsibleUtils::Tags.for installEtcd, setupExternalCA, setupControlplaneVip, kubeadmInit, applyNetwork, kubeadmJoinMaster, kubeadmJoin
        return self
    end
end

class Roles
    Master  = 'Master'    
    Node    = 'Node'
    Etcd    = 'Etcd'
    
    All    = [Master, Node, Etcd]
    
    def self.validate(data)
        if data.kind_of?(Array)
            data.each do |s|
                raise "Invalid role '#{s}'. valid roles are [#{All.join(', ')}]." unless All.include?(s)
            end 
        else
            raise "invalid roles list '#{data}'" 
        end
    end
end 