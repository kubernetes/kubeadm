module Utils

    def self.putsTitle(msg)
        puts "\033[1;34m==> #{msg}:\033[0m"
    end

    def self.putsTitle2(msg)
        puts "\033[1;33m==>    #{msg}:\033[0m"
    end

    class Integer
        def self.validate(string)
            Integer(string) 
        end
    end

    class CIDR
        def self.validate(string)
            unless string =~ /^(?=\d+\.\d+\.\d+\.\d+($|\/))(([1-9]?\d|1\d\d|2[0-4]\d|25[0-5])\.?){4}(\/([0-9]|[1-2][0-9]|3[0-2]))?$/ 
                raise "invalid CIDR '#{string}'" 
            end
        end
    end

    class CIDRList
        def self.validate(data)
            if data.kind_of?(Array)
                data.each do |s|
                    CIDR.validate(s)
                end 
            else
                raise "invalid CIDR list '#{data}'" 
            end
        end
    end 

end