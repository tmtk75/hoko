run:
	go run main.go run

agent:
	serf agent -config-file serf-config.json 

tags:
	serf tags -set webhook=push

hup:
	kill -1 `ps axu | egrep 'serf agent' | egrep -v 'egrep serf agent' | awk '{print $$2}'`

sample:
	curl -v -XPOST -H"x-github-event: push" \
	  localhost:3000/serf/query/github -d '{"ref":"refs/head/master"}'

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

