#
run:
	SECRET_TOKEN=`cat test/secret_token.txt` go run main.go client.go run -d

tags:
	serf tags -set webhook=push

secret:
	@cat test/secret_token.txt

sample:
	curl -v -XPOST \
	  -H"x-hub-signature: `cat test/x-hub-signature.txt`" \
	  localhost:9981/serf/query/hoko \
	  -d @test/webhook-body.json 

hup:
	kill -1 `ps axu | egrep 'serf agent' | egrep -v 'egrep serf agent' | awk '{print $$2}'`

post:
	echo '{"event":"custom"}' | \
	  SECRET_TOKEN=`cat test/secret_token.txt` go run \
	  main.go client.go post

#
#
#
build: gox zip

#
# ansible
#
.py:
	virtualenv .py

# DON'T FORGET source
# $ source .py/bin/activate

user=ec2-user
private_key=~/.ssh/id_rsa
ec2_ipaddr=replace with your host

#user=vagrant
#private_key=~/.vagrant.d/insecure_private_key
#ec2_ipaddr=192.168.111.222

ping: .py/bin/ansible
	ansible -u $(user) -m ping -i provision/hosts --private-key $(private_key) $(ec2_ipaddr)

playbook:
	ansible-playbook -u $(user) -i provision/hosts --private-key $(private_key) provision/playbook.yaml

jq:
	brew install jq

#
# secrity groups
#
vpcs:
	aws ec2 describe-vpcs

vpc-id:
	@aws ec2 describe-vpcs | jq -r ".Vpcs[0].VpcId"

sg-hoko:
	aws ec2 create-security-group --vpc-id $(vpc_id) --group-name "hoko" --description "hoko" > .sg-hoko.json
	aws ec2 create-tags --resources `jq -r .GroupId < .sg-hoko.json` --tags Key=role,Value=hoko
	aws ec2 authorize-security-group-ingress --group-id `jq -r .GroupId < .sg-hoko.json` --port 22   --protocol tcp --cidr 0.0.0.0/0
	aws ec2 authorize-security-group-ingress --group-id `jq -r .GroupId < .sg-hoko.json` --port 9981 --protocol tcp --cidr 0.0.0.0/0

#
# Launch a t2.micro instance on AWS console, which can be only launched on a VPC.
# It's troublesome if you launch it with awscli :D
#
launch-ec2-instance:
	open "https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-1#LaunchInstanceWizard:"

# See to install and setup gox
# https://github.com/mitchellh/gox
gox:
	gox -os="linux darwin" -arch=amd64 -output "pkg/dist/{{.Dir}}_{{.OS}}_{{.Arch}}"

hoko: main.go client.go
	go build

version=`./hoko -v | sed 's/hoko version //g'`

release: hoko
	echo ghr -u tmtk75 v$(version) pkg/dist/hoko_linux_amd64.zip

zip: pkg/dist/hoko_linux_amd64.zip pkg/dist/hoko_darwin_amd64.zip

pkg/dist/hoko_linux_amd64.zip: pkg/dist/hoko_linux_amd64
	cd pkg/dist; mv hoko_linux_amd64 hoko; zip hoko_linux_amd64.zip hoko

pkg/dist/hoko_darwin_amd64.zip: pkg/dist/hoko_darwin_amd64
	cd pkg/dist; mv hoko_darwin_amd64 hoko; zip hoko_darwin_amd64.zip hoko

clean:
	rm -f ssh-config
distclean: clean
	rm -rf hoko pkg

##
ssh-config:
	vagrant ssh-config > ssh-config

galaxy:
	ansible-galaxy install -p roles tmtk75.hoko

vagrant-deploy: ssh-config
	ansible-playbook -i "default," playbook.yaml

##
ansible: .e/bin/ansible
.e/bin/ansible: .e
	.e/bin/pip2.7 install ansible
.e:
	virtualenv .e

