# -*- mode: ruby -*-
Vagrant.configure("2") do |config|

  config.vm.box     = "centos65-x86_64-20140116"
  config.vm.box_url = "https://github.com/2creatives/vagrant-centos/releases/download/v6.5.3/centos65-x86_64-20140116.box"

  config.vm.provider :virtualbox do |vb|
   vb.name = "hoko-vm"
   vb.customize ["modifyvm", :id, "--memory", "1024"]
  end

end
