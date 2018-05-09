require 'fileutils'

class VagrantUtils

    def self.silent
        if ARGV.length >= 1 && ["provision", "up"].include?(ARGV[0].downcase) then 
            return false 
        else 
            return true 
        end
    end

    def self.InjectKubeadmFromBuildOutput(build_output_path:, prefix:nil)
        root = File.join ENV["HOME"], "/go"
        if ENV.has_key?("kube_root") 
            root = ENV["kube_root"]
        elsif ENV.has_key?("GOPATH") 
            root = ENV["GOPATH"]
        end 

        kubeadm_source_binary = File.join root, "/src/k8s.io/kuberneteskubernetes", build_output, "kubeadm"

        spearator = if prefix == nil then "" else "-" end
        kubeadm_target_binary = File.join "/bin", "#{prefix}#{separator}kubeadm"

        FileUtils.cp kubeadm_source_binary, kubeadm_target_binary

        return kubeadm_target_binary
    end
    
end
