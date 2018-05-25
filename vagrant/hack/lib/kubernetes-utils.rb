module KuberentesUtils

    class DNS1123Label
        def self.validate(string)
            unless string =~ /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/ 
                raise "invalid dns1123Label '#{string}'" 
            end
        end
    end
    
    class DNS1123Subdomain
        def self.validate(string)
            unless string =~ /^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$/ 
                raise "invalid dns1123Subdomain '#{string}'" 
            end
        end
    end

end