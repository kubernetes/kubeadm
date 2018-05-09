module AnsibleUtils

    class Inventory
        All = 'all'

        def self.for(machines, roles)
            groups       = {}
            groups[All]  = []

            roles.each do |r|
                role = "#{r.downcase}s"
                groups[role]=[]
            end

            machines.each do |machine|
                groups[All] << machine.name 
                machine.roles.each do |r|
                    role = "#{r.downcase}s"
                    if groups[role].empty? then  groups["primary_#{r.downcase}"]=[machine.name] end
                    groups[role] << machine.name 
                end
            end
            
            return groups
        end
    end

    class Tags
        Prerequisites       = "prereqs" 
        Etcd                = "etcd"
        ExternalCA          = "externalCA"
        ExternalVIP         = "vip"
        Kubeadm_config      = "config"
        Kubeadm_Init        = "init"
        Kubeadm_Network     = "network"
        Kubeadm_JoinMaster  = "joinMaster"
        Kubeadm_Join        = "join"

        def self.for(installEtcd, setupExternalCA, setupControlplaneVip, kubeadmInit, applyNetwork, kubeadmJoinMaster, kubeadmJoin)
            if !kubeadmInit && (applyNetwork || kubeadmJoinMaster || kubeadmJoin) then raise "invalid action configuration: applyNetwork, kubeadmJoinMaster or kubeadmJoin will fail without kubeadmInit." end

            if kubeadmInit && (!installEtcd && @etcdMode == KubeadmOptions::EtcdMode::External) then raise "invalid ansible action configuration: kubeadmInit will fail without installing external ETCD." end
            if kubeadmInit && (!setupExternalCA && @certificateAuthority == KubeadmOptions::CertificateAuthority::External) then raise "invalid ansible action configuration: kubeadmInit will automatically create a local pki if the exxpected external CA is not provisioned." end
            if kubeadmInit && (!setupControlplaneVip && @controlplaneAvailability == KubeadmOptions::ControlplaneAvailability::MultiMaster) then raise "invalid ansible action configuration: kubeadmInit will fail if the controlplane vip is not provisioned." end
        
            ansible_tags = [Prerequisites, Kubeadm_config]
            ansible_tags << Etcd                if installEtcd
            ansible_tags << ExternalCA          if setupExternalCA
            ansible_tags << ExternalVIP         if setupControlplaneVip
            ansible_tags << Kubeadm_Init        if kubeadmInit
            ansible_tags << Network             if applyNetwork
            ansible_tags << Kubeadm_JoinMaster  if kubeadmJoinMaster
            ansible_tags << Kubeadm_Join        if kubeadmJoin
    
            return ansible_tags
        end
    end

end