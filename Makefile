run:
	SECRET_TOKEN=`cat test/secret_token.txt` go run main.go run

agent:
	serf agent -config-file serf-config.json 

tags:
	serf tags -set webhook=push

hup:
	kill -1 `ps axu | egrep 'serf agent' | egrep -v 'egrep serf agent' | awk '{print $$2}'`

secret:
	@cat test/secret_token.txt

sample:
	curl -v -XPOST \
	  -H"x-hub-signature: `cat test/x-hub-signature.txt`" \
	  localhost:3000/serf/query/github \
	  -d @test/webhook-body.json 

sample-sh:
	curl -v -XPOST -H"x-github-event: push" \
	  localhost:3000/ -d '{"ref":"refs/head/master"}'

#
# ansible
#
.py:
	virtualenv .py

#.py/bin/ansible:
#	. .py/bin/activate
#	pip install .py
#

ping: .py/bin/ansible
	ansible \
	  -u ec2-user -m ping -i hosts --private-key ~/.ssh/id_rsa \
	  $(ec2_ipaddr)

playbook:
	ansible-playbook  \
	  -u ec2-user -i hosts --private-key ~/.ssh/id_rsa \
	  provision.yaml

jq:
	brew install jq

#
#
#
vpcs:
	aws ec2 describe-vpcs

vpc-id:
	@aws ec2 describe-vpcs | jq -r ".Vpcs[0].VpcId"

sg-hoko:
	aws ec2 create-security-group --vpc-id $(vpc_id) --group-name "hoko" --description "hoko" > .sg-hoko.json
	aws ec2 create-tags --resources `jq -r .GroupId < .sg-hoko.json` --tags Key=role,Value=hoko
	aws ec2 authorize-security-group-ingress --group-id `jq -r .GroupId < .sg-hoko.json` --port 22   --protocol tcp --cidr 0.0.0.0/0
	aws ec2 authorize-security-group-ingress --group-id `jq -r .GroupId < .sg-hoko.json` --port 3000 --protocol tcp --cidr 0.0.0.0/0

#
# Launch a t2.micro instance on AWS console, which can be only launched on a VPC.
# It's troublesome if you launch it with awscli :D
#
launch-ec2-instance:
	open "https://console.aws.amazon.com/ec2/v2/home?region=ap-northeast-1#LaunchInstanceWizard:"
