require_relative 'kubeadm-options.rb'
require_relative 'vagrant-utils.rb'

class TestGuide

    def self.print(c)
        if VagrantUtils.silent then return end

        Utils.putsTitle "Test guide for #{c.name}"

        i = 1; print_overview i, c

        if c.etcdMode == KubeadmOptions::EtcdMode::External || 
            c.certificateAuthority == KubeadmOptions::CertificateAuthority::External ||
            c.controlplaneAvailability == KubeadmOptions::ControlplaneAvailability::MultiMaster then
            
            i += 1; print_dependencies i, c
        end

        if (c.etcdMode == KubeadmOptions::EtcdMode::External && !c.ansible_installEtcd) || 
            (c.certificateAuthority == KubeadmOptions::CertificateAuthority::External && !c.ansible_setupExternalCA) ||
            (c.controlplaneAvailability == KubeadmOptions::ControlplaneAvailability::MultiMaster && !c.ansible_setupControlplaneVip) || 
            !c.ansible_kubeadmInit ||
            (c.masters.length>1 && !c.ansible_kubeadmJoinMaster) ||
            (c.nodes.length>1 && !c.ansible_kubeadmJoin) then
            
            i += 1; print_completeprovisioning i, c
        end

        i += 1; print_testplan i, c
    end

    def self.print_overview(i, c)
        Utils.putsTitle "#{i}. Cluster overview"
        puts "    The cluster is composed by following machines:"
        puts "    - #{c.masters.length} master nodes:"
        c.masters.each do |m|
            puts "       - #{m.name}"
        end
        if c.nodes.length > 0 
            puts "    - #{c.nodes.length} nodes:" 
            c.nodes.each do |m|
                puts "       - #{m.name}"
            end
        end
    end

    def self.print_dependencies(i, c)
        Utils.putsTitle "#{i}. Cluster dependencies"
        verb = if c.ansible_kubeadmInit then "uses" else "will use" end
     
        if c.controlplaneAvailability == KubeadmOptions::ControlplaneAvailability::MultiMaster then 
            if c.ansible_setupControlplaneVip
                puts "    - The cluster #{verb} an external VIP for the controlplane to be configured on all the Master machines."
            else
                puts "    - The cluster #{verb} an external VIP for the controlplane alerady configured on all the Master machines."
            end
        end 

        if c.certificateAuthority == KubeadmOptions::CertificateAuthority::External then 
            if c.ansible_setupExternalCA
                puts "    - The cluster #{verb} an external Certificate Authority to be created and distributed on all the Master machines."
            else
                puts "    - The cluster #{verb} an external Certificate Authority alerady created and distributed on all the Master machines."
            end
        end  

        if c.etcdMode == KubeadmOptions::EtcdMode::External then 
            if c.ansible_installEtcd
                puts "    - The cluster #{verb} an external etcd cluster to be deployed on following machines:"
            else
                puts "    - The cluster #{verb} an external etcd cluster alerady deployed on following machines:"
            end
            c.etcds.each do |m|
                puts "       - #{m.name}"
            end
        end 

    end
    
    def self.print_completeprovisioning(i, c)
        Utils.putsTitle "#{i}. Complete cluster provisioning"
        
        j = 0

        if c.etcdMode == KubeadmOptions::EtcdMode::External && !c.ansible_installEtcd
            j += 1; Utils.putsTitle2 "#{j}. Install Etcd"
            puts "    - Install etcd on etcd machines using your preferred approach."
            puts "    - Ensure that etcd endpoint and eventually etcd TLS certificates are set in kubeadm.conf"
            puts
        end

        if c.certificateAuthority == KubeadmOptions::CertificateAuthority::External && !c.ansible_setupExternalCA
            j += 1; Utils.putsTitle2 "#{j}. Setup an external Certificate Authority"
            puts "    - Create an external CA using your preferred approach."
            puts "    - Copy the certificates on `/etc/kubernetes/pki` all master nodes."
            puts
        end
        
        if c.controlplaneAvailability == KubeadmOptions::ControlplaneAvailability::MultiMaster && !c.ansible_setupControlplaneVip
            j += 1; Utils.putsTitle2 "#{j}. Setup an external VIP or a loab balancer for the controlplane"
            puts "    - Create an external VIP/load balancer using your preferred approach."
            puts "    - Ensure that the controlplane address is set in kubeadm.conf."
            puts
        end 
        
        if !c.ansible_kubeadmInit
            j += 1; Utils.putsTitle2 "#{j}. Kubeadm init"
            puts "    - Run 'Kubeadm init' on #{c.masters.first.name}."
            puts
        end
        
        if !c.ansible_applyNetwork
            j += 1; Utils.putsTitle2 "#{j}. Install network addon"
            puts "    - Run 'kubectl apply ...'' "
            puts
        end

        if c.masters.length>1 && !c.ansible_kubeadmJoinMaster
            j += 1; Utils.putsTitle2 "#{j}. Join additional master nodes"
            puts "    - Run 'Kubeadm join --experimentalmaster' on:"
            c.masters.drop(1).each do |m|
                puts "       - #{m.name}"
            end
            puts
        end

        if c.nodes.length>1 && !c.ansible_kubeadmJoin
            j += 1; Utils.putsTitle2 "#{j}. Join worker nodes"
            puts "    - Run 'Kubeadm join' on:"
            c.nodes.each do |m|
                puts "       - #{m.name}"
            end
            puts
        end
        
        j = 0
    end

    def self.print_testplan(i, c)
        Utils.putsTitle "#{i}. Test Kubeadm and Kubernets"
        
        j = 0

        j += 1; Utils.putsTitle2 "#{j}. Test Kubernetes"
        puts "    'vagrant ssh' to one Master machine and:"
        puts "    - run kubeadm E2E test!"
        puts "    - run kubernetes E2E test!"
        puts "    - use kubectl!" 
        puts

        j += 1; Utils.putsTitle2 "#{j}. Test kubeadm updates"
        puts "    ... " 
        puts

        j += 1; Utils.putsTitle2 "#{j}. Test kubeadm reset"
        puts "    ... " 
        puts
    end
end